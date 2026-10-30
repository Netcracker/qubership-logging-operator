package forwarderaggregator

import (
	"strings"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const storageVolume = "storage"

// AggregatorStatefulSet returns the aggregator StatefulSet that receives logs from the
// forwarders and sends them to the outputs. The caller checks that
// cr.Spec.Fluentbit.Aggregator is set.
func AggregatorStatefulSet(cr *loggingService.LoggingService, cfg *defaults.Defaults) *appsv1.StatefulSet {
	spec := cr.Spec.Fluentbit.Aggregator
	def := cfg.Aggregator
	openshift := cr.Spec.OpenshiftDeploy
	bindVolume := spec.Volume != nil && spec.Volume.Bind

	var volumes []corev1.Volume
	if !bindVolume {
		volumes = append(volumes, emptyDirVolume(storageVolume, resource.MustParse(spec.EffectiveStorageSizeLimit())))
	}
	volumes = append(volumes, secretVolume(configVolume, aggregatorName), emptyDirVolume(tmpVolume, tmpVolumeSize))
	mounts := []corev1.VolumeMount{
		{Name: storageVolume, MountPath: "/fluent-bit/storage"},
		{Name: configVolume, MountPath: configDir, ReadOnly: true},
		{Name: tmpVolume, MountPath: "/tmp"},
	}
	tlsVolumes, tlsMounts := collectorTLSMounts(spec.TLS)
	outputVolumes, outputMounts := outputTLSMounts(spec.Output)
	volumes = append(append(volumes, tlsVolumes...), outputVolumes...)
	mounts = append(append(mounts, tlsMounts...), outputMounts...)

	aggregator := builder.NewContainer(aggregatorName, builder.ContainerOpts{
		Image: spec.DockerImage,
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: httpPort, Protocol: corev1.ProtocolTCP},
			{Name: "log-to-metrics", ContainerPort: logToMetricsPort, Protocol: corev1.ProtocolTCP},
			{Name: "forward", ContainerPort: forwardPort, Protocol: corev1.ProtocolTCP},
		},
		Resources:       *builder.FirstNonNil(spec.Resources, &def.AggregatorResources).DeepCopy(),
		VolumeMounts:    mounts,
		LivenessProbe:   httpProbe("/", 5),
		ReadinessProbe:  httpProbe("/api/v1/health", 5),
		SecurityContext: nonRootSecurityContext(),
	})
	setMainContainerDefaults(&aggregator)

	podSpec := corev1.PodSpec{
		Volumes:                       volumes,
		Containers:                    []corev1.Container{reloadContainer(spec.ConfigmapReload, def.ConfigmapReloadResources), aggregator},
		TerminationGracePeriodSeconds: ptr.To(int64(30)),
		ServiceAccountName:            aggregatorName,
		DeprecatedServiceAccount:      aggregatorName,
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup:             ptr.To(nonRootID),
			FSGroupChangePolicy: ptr.To(corev1.FSGroupChangeOnRootMismatch),
			SeccompProfile:      &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			RunAsNonRoot:        ptr.To(true),
			RunAsUser:           ptr.To(nonRootID),
			RunAsGroup:          ptr.To(nonRootID),
		},
		SchedulerName: corev1.DefaultSchedulerName,
		NodeSelector:  builder.NodeSelector(spec.NodeSelectorKey, spec.NodeSelectorValue),
		Tolerations:   spec.Tolerations,
		Affinity:      spec.Affinity,
	}
	if strings.TrimSpace(spec.PriorityClassName) != "" {
		podSpec.PriorityClassName = spec.PriorityClassName
	}

	replicas := def.AggregatorReplicas
	if spec.Replicas != 0 {
		replicas = int32(spec.Replicas)
	}
	statefulSet := builder.NewStatefulSet(aggregatorName, cr.GetNamespace(), builder.StatefulSetOpts{
		Replicas:    ptr.To(replicas),
		ServiceName: aggregatorName,
		Selector: map[string]string{
			"name":      aggregatorName,
			"component": aggregatorName,
			"provider":  provider(openshift),
		},
		PodSpec:        podSpec,
		Labels:         workloadLabels(aggregatorName, openshift),
		PodLabels:      workloadLabels(aggregatorName, openshift),
		Annotations:    spec.Annotations,
		PodAnnotations: spec.Annotations,
	})
	statefulSet.Spec.Template.Name = aggregatorName
	statefulSet.Spec.RevisionHistoryLimit = ptr.To(int32(10))
	if bindVolume {
		statefulSet.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{storageClaim(spec.Volume, def)}
	}
	util.SetLabelsForWorkload(statefulSet, &statefulSet.Spec.Template.Labels, util.LabelInput{
		Name:            aggregatorName,
		Component:       util.ComponentBackend,
		Instance:        util.GetInstanceLabel(aggregatorName, cr.GetNamespace()),
		Version:         util.GetTagFromImage(spec.DockerImage),
		Technology:      technology,
		ComponentLabels: spec.Labels,
	})
	return statefulSet
}

// AggregatorService returns the Service the forwarders send logs to. The caller checks
// that cr.Spec.Fluentbit.Aggregator is set.
func AggregatorService(cr *loggingService.LoggingService, _ *defaults.Defaults) *corev1.Service {
	openshift := cr.Spec.OpenshiftDeploy
	port := func(name string, number int32) corev1.ServicePort {
		return corev1.ServicePort{Name: name, Protocol: corev1.ProtocolTCP, Port: number, TargetPort: intstr.FromInt32(number)}
	}
	service := builder.NewService(aggregatorName, cr.GetNamespace(), builder.ServiceOpts{
		Type: corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{
			port("logging-fluentbit", httpPort),
			port("log-to-metrics", logToMetricsPort),
			port(aggregatorName, forwardPort),
		},
		Selector: map[string]string{"name": aggregatorName, "provider": provider(openshift)},
		Labels:   map[string]string{"component": aggregatorName, "provider": provider(openshift)},
	})
	util.SetLabelsForResource(service, util.LabelInput{
		Name:            aggregatorName,
		Component:       util.ComponentBackend,
		ComponentLabels: cr.Spec.Fluentbit.Aggregator.Labels,
	}, nil)
	return service
}

// storageClaim returns the claim template of the aggregator buffer volume.
func storageClaim(volume *loggingService.Volume, def defaults.FluentbitAggregatorDefaults) corev1.PersistentVolumeClaim {
	size := def.AggregatorVolumeSize
	if volume.StorageSize != "" {
		size = resource.MustParse(volume.StorageSize)
	}
	claim := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: storageVolume},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: size},
			},
		},
	}
	if volume.StorageClassName != "" {
		claim.Spec.StorageClassName = ptr.To(volume.StorageClassName)
	}
	return claim
}

// outputTLSMounts returns the volumes and mounts of the TLS files for the enabled Loki,
// HTTP, and OpenTelemetry outputs.
func outputTLSMounts(output *loggingService.OutputFluentbit) ([]corev1.Volume, []corev1.VolumeMount) {
	if output == nil {
		return nil, nil
	}
	var volumes []corev1.Volume
	var mounts []corev1.VolumeMount
	add := func(name string, certs loggingService.Certificates) {
		ca, cert, key := certificateFiles(certs)
		v, m := certificateMounts(name+"-tls", "/fluent-bit/output/"+name+"/tls", ca, cert, key)
		volumes = append(volumes, v...)
		mounts = append(mounts, m...)
	}
	if output.Loki != nil && output.Loki.Enabled && output.Loki.TLS != nil && output.Loki.TLS.Enabled {
		add("loki", output.Loki.TLS.Certificates)
	}
	if output.Http != nil && output.Http.Enabled && output.Http.TLS != nil && output.Http.TLS.Enabled {
		add("http", output.Http.TLS.Certificates)
	}
	if output.Otel != nil && output.Otel.Enabled && output.Otel.TLS != nil && output.Otel.TLS.Enabled {
		add("otel", output.Otel.TLS.Certificates)
	}
	return volumes, mounts
}
