package forwarderaggregator

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

func aggregatorSelector(openshift bool) map[string]string {
	return map[string]string{
		"name":      AggregatorName,
		"component": AggregatorName,
		"provider":  providerLabel(openshift),
	}
}

func buildAggregatorService(cr *loggingService.LoggingService) *corev1.Service {
	openshift := cr.Spec.OpenshiftDeploy
	httpPort := int32(2020)
	logToMetrics := int32(2021)
	forwardPort := int32(24224)
	svc := build.NewService(AggregatorName, cr.GetNamespace(), AggregatorName, build.ServiceOpts{
		Type: corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{
			{Name: "logging-fluentbit", Port: httpPort, TargetPort: intstr.FromInt32(httpPort), Protocol: corev1.ProtocolTCP},
			{Name: "log-to-metrics", Port: logToMetrics, TargetPort: intstr.FromInt32(logToMetrics), Protocol: corev1.ProtocolTCP},
			{Name: AggregatorName, Port: forwardPort, TargetPort: intstr.FromInt32(forwardPort), Protocol: corev1.ProtocolTCP},
		},
		Selector: map[string]string{
			"name":     AggregatorName,
			"provider": providerLabel(openshift),
		},
		ExtraLabels: resourceProviderLabels(AggregatorName, openshift),
	})
	util.SetLabelsForResource(svc, util.LabelInput{
		Name:            AggregatorName,
		Component:       "fluentbit",
		ComponentLabels: cr.Spec.Fluentbit.Aggregator.Labels,
	}, nil)
	return svc
}

func buildAggregatorStatefulSet(cr *loggingService.LoggingService, def config.FluentbitAggregatorDefaults) *appsv1.StatefulSet {
	agg := cr.Spec.Fluentbit.Aggregator
	openshift := cr.Spec.OpenshiftDeploy

	containers := []corev1.Container{
		buildAggregatorConfigmapReload(agg, def),
		buildAggregatorMain(cr, def),
	}
	volumes := buildAggregatorVolumes(cr)
	terminationGrace := def.AggregatorTGracePeriodS

	pod := corev1.PodSpec{
		TerminationGracePeriodSeconds: &terminationGrace,
		ServiceAccountName:            AggregatorName,
		Tolerations:                   agg.Tolerations,
		NodeSelector:                  nodeSelector(agg.NodeSelectorKey, agg.NodeSelectorValue),
		Affinity:                      agg.Affinity,
		PriorityClassName:             agg.PriorityClassName,
		Volumes:                       volumes,
		Containers:                    containers,
		SchedulerName:                 "default-scheduler",
		SecurityContext:               def.AggregatorPodSecurityContext.DeepCopy(),
	}

	replicas := def.AggregatorReplicas
	if agg.Replicas != 0 {
		replicas = int32(agg.Replicas)
	}
	revHistory := def.AggregatorRevisionHistory

	ss := build.NewStatefulSet(AggregatorName, cr.GetNamespace(), "fluentbit", build.StatefulSetOpts{
		Replicas:    &replicas,
		ServiceName: AggregatorName,
		Selector:    aggregatorSelector(openshift),
		UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
			Type: appsv1.RollingUpdateStatefulSetStrategyType,
		},
		PodSpec:              pod,
		PodName:              AggregatorName,
		PodLabels:            clusterServicePodLabel(aggregatorSelector(openshift)),
		PodAnnotations:       agg.Annotations,
		VolumeClaimTemplates: aggregatorPVCs(agg, def),
		RevisionHistoryLimit: &revHistory,
		ExtraLabels:          clusterServicePodLabel(aggregatorSelector(openshift)),
		Annotations:          agg.Annotations,
	})
	util.SetLabelsForWorkload(ss, &ss.Spec.Template.Labels, util.LabelInput{
		Name:            AggregatorName,
		Component:       "fluentbit",
		Instance:        util.GetInstanceLabel(AggregatorName, cr.GetNamespace()),
		Version:         util.GetTagFromImage(agg.DockerImage),
		Technology:      Technology,
		ComponentLabels: agg.Labels,
	})
	return ss
}

// aggregatorPVCs returns the volumeClaimTemplates when Volume.Bind is set, otherwise
// nil (the matching pod-level emptyDir is added by buildAggregatorVolumes).
func aggregatorPVCs(agg *loggingService.FluentbitAggregator, def config.FluentbitAggregatorDefaults) []corev1.PersistentVolumeClaim {
	if agg.Volume == nil || !agg.Volume.Bind {
		return nil
	}
	size := agg.Volume.StorageSize
	if size == "" {
		size = def.AggregatorPVCStorageSize
	}
	q := resource.MustParse(size)
	return []corev1.PersistentVolumeClaim{{
		ObjectMeta: metaName("storage"),
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: stringPtrOrNil(agg.Volume.StorageClassName),
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: q,
				},
			},
		},
	}}
}

func buildAggregatorConfigmapReload(agg *loggingService.FluentbitAggregator, def config.FluentbitAggregatorDefaults) corev1.Container {
	image := ""
	resources := def.AggregatorConfigmapReloadRes.DeepCopy()
	if agg.ConfigmapReload != nil {
		image = agg.ConfigmapReload.DockerImage
		if agg.ConfigmapReload.Resources != nil {
			resources = agg.ConfigmapReload.Resources.DeepCopy()
		}
	}
	allowEsc := false
	return build.NewContainer(ReloadContainer, build.ContainerOpts{
		Image: image,
		Args:  def.ConfigmapReloadArgs,
		Ports: []corev1.ContainerPort{
			{Name: "reload-metrics", ContainerPort: def.ConfigmapReloadPort},
		},
		Resources: *resources,
		VolumeMounts: []corev1.VolumeMount{
			{Name: "config", MountPath: "/fluent-bit/etc", ReadOnly: true},
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: &allowEsc,
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
	})
}

func buildAggregatorMain(cr *loggingService.LoggingService, def config.FluentbitAggregatorDefaults) corev1.Container {
	agg := cr.Spec.Fluentbit.Aggregator
	resources := def.AggregatorResources.DeepCopy()
	if agg.Resources != nil {
		resources = agg.Resources.DeepCopy()
	}
	return build.NewContainer(AggregatorName, build.ContainerOpts{
		Image:           agg.DockerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             aggregatorEnv(agg),
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: def.HTTPPort, Protocol: corev1.ProtocolTCP},
			{Name: "log-to-metrics", ContainerPort: def.LogToMetricsPort, Protocol: corev1.ProtocolTCP},
			{Name: "forward", ContainerPort: def.ForwardPort, Protocol: corev1.ProtocolTCP},
		},
		LivenessProbe:   def.AggregatorLivenessProbe.DeepCopy(),
		ReadinessProbe:  def.AggregatorReadinessProbe.DeepCopy(),
		Resources:       *resources,
		VolumeMounts:    buildAggregatorMounts(agg),
		SecurityContext: def.AggregatorContainerSecCtx.DeepCopy(),
	})
}

func aggregatorEnv(agg *loggingService.FluentbitAggregator) []corev1.EnvVar {
	if agg.Output == nil {
		return nil
	}
	var env []corev1.EnvVar
	if agg.Output.Loki != nil && agg.Output.Loki.Enabled {
		env = append(env, authEnv("LOKI", agg.Output.Loki.Auth)...)
	}
	if agg.Output.Http != nil && agg.Output.Http.Enabled {
		env = append(env, authEnv("HTTP", agg.Output.Http.Auth)...)
	}
	if agg.Output.Otel != nil && agg.Output.Otel.Enabled {
		env = append(env, authEnv("OTEL", agg.Output.Otel.Auth)...)
	}
	return env
}

func buildAggregatorVolumes(cr *loggingService.LoggingService) []corev1.Volume {
	agg := cr.Spec.Fluentbit.Aggregator
	mode := configMapDefaultMode
	tokenMode := authTokenDefaultMode
	var vols []corev1.Volume
	// storage: emptyDir unless a PVC is bound (in which case the volumeClaimTemplate
	// supplies the volume of the same name).
	if agg.Volume == nil || !agg.Volume.Bind {
		vols = append(vols, corev1.Volume{
			Name:         "storage",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		})
	}
	vols = append(vols, corev1.Volume{
		Name: "config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: AggregatorName},
				DefaultMode:          &mode,
			},
		},
	})
	vols = append(vols, tlsVolumes(&agg.TLS)...)
	if agg.Output != nil {
		if agg.Output.Loki != nil && agg.Output.Loki.Enabled {
			if agg.Output.Loki.Auth != nil && isSecretKeySet(agg.Output.Loki.Auth.Token) {
				vols = append(vols, secretVolume("loki-auth-token", agg.Output.Loki.Auth.Token.Name, tokenMode))
			}
			if agg.Output.Loki.TLS != nil && agg.Output.Loki.TLS.Enabled {
				vols = append(vols, outputTLSVolumes("loki", agg.Output.Loki.TLS.Certificates)...)
			}
		}
		if agg.Output.Http != nil && agg.Output.Http.Enabled && agg.Output.Http.TLS != nil && agg.Output.Http.TLS.Enabled {
			vols = append(vols, outputTLSVolumes("http", agg.Output.Http.TLS.Certificates)...)
		}
		if agg.Output.Otel != nil && agg.Output.Otel.Enabled && agg.Output.Otel.TLS != nil && agg.Output.Otel.TLS.Enabled {
			vols = append(vols, outputTLSVolumes("otel", agg.Output.Otel.TLS.Certificates)...)
		}
	}
	return vols
}

func buildAggregatorMounts(agg *loggingService.FluentbitAggregator) []corev1.VolumeMount {
	storageMount := corev1.VolumeMount{Name: "storage", MountPath: "/fluent-bit/storage"}
	if agg.Volume != nil && agg.Volume.Bind {
		// Legacy asset sets readOnly: false explicitly on this branch; struct field
		// defaults to false either way so the field is omitted intentionally.
		storageMount.ReadOnly = false
	}
	mounts := []corev1.VolumeMount{
		storageMount,
		{Name: "config", MountPath: "/fluent-bit/etc", ReadOnly: true},
	}
	mounts = append(mounts, tlsVolumeMounts(&agg.TLS)...)
	if agg.Output != nil {
		if agg.Output.Loki != nil && agg.Output.Loki.Enabled && agg.Output.Loki.TLS != nil && agg.Output.Loki.TLS.Enabled {
			mounts = append(mounts, outputTLSMounts("loki", "/fluent-bit/output/loki/tls", agg.Output.Loki.TLS.Certificates)...)
		}
		if agg.Output.Http != nil && agg.Output.Http.Enabled && agg.Output.Http.TLS != nil && agg.Output.Http.TLS.Enabled {
			mounts = append(mounts, outputTLSMounts("http", "/fluent-bit/output/http/tls", agg.Output.Http.TLS.Certificates)...)
		}
		if agg.Output.Otel != nil && agg.Output.Otel.Enabled && agg.Output.Otel.TLS != nil && agg.Output.Otel.TLS.Enabled {
			mounts = append(mounts, outputTLSMounts("otel", "/fluent-bit/output/otel/tls", agg.Output.Otel.TLS.Certificates)...)
		}
	}
	return mounts
}

// metaName returns a minimal ObjectMeta with only the given Name. PVC templates need
// only metadata.name; the StatefulSet controller fills in namespace + owner refs.
func metaName(name string) metav1ObjectMeta {
	return metav1ObjectMeta{Name: name}
}
