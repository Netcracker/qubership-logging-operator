package forwarderaggregator

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

func forwarderSelector(openshift bool) map[string]string {
	return map[string]string{
		"component": ForwarderName,
		"provider":  providerLabel(openshift),
	}
}

func buildForwarderService(cr *loggingService.LoggingService) *corev1.Service {
	openshift := cr.Spec.OpenshiftDeploy
	port := int32(2020)
	svc := build.NewService(ForwarderName, cr.GetNamespace(), ForwarderName, build.ServiceOpts{
		Type: corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{
			{Name: ForwarderName, Port: port, TargetPort: intstr.FromInt32(port), Protocol: corev1.ProtocolTCP},
		},
		Selector: map[string]string{
			"name":     ForwarderName,
			"provider": providerLabel(openshift),
		},
		ExtraLabels: resourceProviderLabels(ForwarderName, openshift),
	})
	util.SetLabelsForResource(svc, util.LabelInput{
		Name:            ForwarderName,
		Component:       "fluentbit",
		ComponentLabels: cr.Spec.Fluentbit.Labels,
	}, nil)
	return svc
}

func buildForwarderDaemonSet(cr *loggingService.LoggingService, def config.FluentbitAggregatorDefaults) *appsv1.DaemonSet {
	spec := cr.Spec.Fluentbit
	openshift := cr.Spec.OpenshiftDeploy

	containers := []corev1.Container{
		buildForwarderConfigmapReload(spec, def),
		buildForwarderMain(cr, def),
	}
	volumes := buildForwarderVolumes(cr)
	terminationGrace := def.ForwarderTGracePeriodS

	pod := corev1.PodSpec{
		TerminationGracePeriodSeconds: &terminationGrace,
		ServiceAccountName:            ForwarderServiceAccount,
		Tolerations:                   spec.Tolerations,
		NodeSelector:                  nodeSelector(spec.NodeSelectorKey, spec.NodeSelectorValue),
		Affinity:                      spec.Affinity,
		PriorityClassName:             spec.PriorityClassName,
		Volumes:                       volumes,
		Containers:                    containers,
		SchedulerName:                 "default-scheduler",
	}

	maxUnavailable := def.ForwarderMaxUnavailable
	ds := build.NewDaemonSet(ForwarderName, cr.GetNamespace(), "fluentbit", build.DaemonSetOpts{
		Selector:        forwarderSelector(openshift),
		MinReadySeconds: def.ForwarderMinReadyS,
		UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
			Type: appsv1.RollingUpdateDaemonSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: &maxUnavailable,
			},
		},
		PodSpec:        pod,
		PodName:        ForwarderName,
		PodLabels:      clusterServicePodLabel(forwarderSelector(openshift)),
		PodAnnotations: spec.Annotations,
		ExtraLabels:    clusterServicePodLabel(forwarderSelector(openshift)),
		Annotations:    spec.Annotations,
	})

	util.SetLabelsForWorkload(ds, &ds.Spec.Template.Labels, util.LabelInput{
		Name:            ForwarderName,
		Component:       "fluentbit",
		Instance:        util.GetInstanceLabel(ForwarderName, cr.GetNamespace()),
		Version:         util.GetTagFromImage(spec.DockerImage),
		Technology:      Technology,
		ComponentLabels: spec.Labels,
	})
	return ds
}

func buildForwarderConfigmapReload(spec *loggingService.Fluentbit, def config.FluentbitAggregatorDefaults) corev1.Container {
	image := ""
	var resources corev1.ResourceRequirements
	if spec.ConfigmapReload != nil {
		image = spec.ConfigmapReload.DockerImage
		if spec.ConfigmapReload.Resources != nil {
			resources = *spec.ConfigmapReload.Resources.DeepCopy()
		}
	}
	allowEsc := false
	return build.NewContainer(ReloadContainer, build.ContainerOpts{
		Image: image,
		Args:  def.ConfigmapReloadArgs,
		Ports: []corev1.ContainerPort{
			{Name: "reload-metrics", ContainerPort: def.ConfigmapReloadPort},
		},
		Resources: resources,
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

func buildForwarderMain(cr *loggingService.LoggingService, def config.FluentbitAggregatorDefaults) corev1.Container {
	spec := cr.Spec.Fluentbit
	var resources corev1.ResourceRequirements
	if spec.Resources != nil {
		resources = *spec.Resources.DeepCopy()
	}
	return build.NewContainer(ForwarderName, build.ContainerOpts{
		Image:           spec.DockerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{
				Name: "NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
				},
			},
		},
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: def.HTTPPort, Protocol: corev1.ProtocolTCP},
		},
		LivenessProbe:   def.ForwarderLivenessProbe.DeepCopy(),
		ReadinessProbe:  def.ForwarderReadinessProbe.DeepCopy(),
		Resources:       resources,
		VolumeMounts:    buildForwarderMounts(cr),
		SecurityContext: &corev1.SecurityContext{Privileged: boolPtr(spec.SecurityContextPrivileged)},
	})
}

func buildForwarderVolumes(cr *loggingService.LoggingService) []corev1.Volume {
	spec := cr.Spec.Fluentbit
	mode := configMapDefaultMode
	var vols []corev1.Volume
	vols = append(vols, dockerHostPathVolumes(cr.Spec.ContainerRuntimeType, cr.Spec.OSKind)...)
	vols = append(vols,
		corev1.Volume{
			Name: "varlog",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: "/var/log", Type: hostPathType("")},
			},
		},
		corev1.Volume{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: ForwarderName},
					DefaultMode:          &mode,
				},
			},
		},
	)
	vols = append(vols, tlsVolumes(&spec.TLS)...)
	if len(spec.AdditionalVolumes) > 0 {
		vols = append(vols, spec.AdditionalVolumes...)
	}
	return vols
}

func buildForwarderMounts(cr *loggingService.LoggingService) []corev1.VolumeMount {
	spec := cr.Spec.Fluentbit
	var mounts []corev1.VolumeMount
	mounts = append(mounts, dockerVolumeMounts(cr.Spec.ContainerRuntimeType, cr.Spec.OSKind)...)
	mounts = append(mounts,
		corev1.VolumeMount{Name: "varlog", MountPath: "/var/log"},
		corev1.VolumeMount{Name: "config", MountPath: "/fluent-bit/etc", ReadOnly: true},
	)
	mounts = append(mounts, tlsVolumeMounts(&spec.TLS)...)
	if len(spec.AdditionalVolumeMounts) > 0 {
		mounts = append(mounts, spec.AdditionalVolumeMounts...)
	}
	return mounts
}
