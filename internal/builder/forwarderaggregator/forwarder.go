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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

// ForwarderDaemonSet returns the forwarder DaemonSet that reads node logs and sends
// them to the aggregator. The caller checks that cr.Spec.Fluentbit is set and sets
// cr.Spec.ContainerRuntimeType to the detected container runtime.
func ForwarderDaemonSet(cr *loggingService.LoggingService, cfg *defaults.Defaults) *appsv1.DaemonSet {
	spec := cr.Spec.Fluentbit
	def := cfg.Aggregator
	openshift := cr.Spec.OpenshiftDeploy

	volumes, mounts := forwarderStorage(cr)
	tlsVolumes, tlsMounts := collectorTLSMounts(spec.TLS)
	volumes = append(volumes, tlsVolumes...)
	mounts = append(mounts, tlsMounts...)

	forwarder := builder.NewContainer(forwarderName, builder.ContainerOpts{
		Image:           spec.DockerImage,
		SecurityContext: forwarderSecurityContext(spec.SecurityContextPrivileged),
		Ports:           []corev1.ContainerPort{{Name: "http", ContainerPort: httpPort, Protocol: corev1.ProtocolTCP}},
		Env: []corev1.EnvVar{{
			Name:      "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}},
		}},
		Resources:      *builder.FirstNonNil(spec.Resources, &def.ForwarderResources).DeepCopy(),
		VolumeMounts:   mounts,
		LivenessProbe:  httpProbe("/", 3),
		ReadinessProbe: httpProbe("/api/v1/health", 5),
	})
	setMainContainerDefaults(&forwarder)

	podSpec := corev1.PodSpec{
		SecurityContext:               forwarderPodSecurityContext(openshift),
		Volumes:                       volumes,
		Containers:                    []corev1.Container{reloadContainer(spec.ConfigmapReload, def.ConfigmapReloadResources), forwarder},
		TerminationGracePeriodSeconds: ptr.To(int64(30)),
		ServiceAccountName:            util.FluentbitComponentName,
		SchedulerName:                 corev1.DefaultSchedulerName,
		NodeSelector:                  builder.NodeSelector(spec.NodeSelectorKey, spec.NodeSelectorValue),
		Tolerations:                   spec.Tolerations,
		Affinity:                      spec.Affinity,
	}
	if strings.TrimSpace(spec.PriorityClassName) != "" {
		podSpec.PriorityClassName = spec.PriorityClassName
	}

	maxUnavailable := intstr.FromInt32(1)
	daemonSet := builder.NewDaemonSet(forwarderName, cr.GetNamespace(), builder.DaemonSetOpts{
		Selector:        map[string]string{"component": forwarderName, "provider": provider(openshift)},
		MinReadySeconds: 60,
		UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
			Type:          appsv1.RollingUpdateDaemonSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{MaxUnavailable: &maxUnavailable},
		},
		PodSpec:        podSpec,
		Labels:         workloadLabels(forwarderName, openshift),
		PodLabels:      workloadLabels(forwarderName, openshift),
		Annotations:    spec.Annotations,
		PodAnnotations: spec.Annotations,
	})
	daemonSet.Spec.Template.Name = forwarderName
	util.SetLabelsForWorkload(daemonSet, &daemonSet.Spec.Template.Labels, util.LabelInput{
		Name:            forwarderName,
		Component:       util.ComponentBackend,
		Instance:        util.GetInstanceLabel(forwarderName, cr.GetNamespace()),
		Version:         util.GetTagFromImage(spec.DockerImage),
		Technology:      technology,
		ComponentLabels: spec.Labels,
	})
	return daemonSet
}

// ForwarderService returns the Service in front of the forwarder pods. The caller
// checks that cr.Spec.Fluentbit is set.
func ForwarderService(cr *loggingService.LoggingService, _ *defaults.Defaults) *corev1.Service {
	openshift := cr.Spec.OpenshiftDeploy
	service := builder.NewService(forwarderName, cr.GetNamespace(), builder.ServiceOpts{
		Type: corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{{
			Name:       forwarderName,
			Protocol:   corev1.ProtocolTCP,
			Port:       httpPort,
			TargetPort: intstr.FromInt32(httpPort),
		}},
		Selector: map[string]string{"name": forwarderName, "provider": provider(openshift)},
		Labels:   map[string]string{"component": forwarderName, "provider": provider(openshift)},
	})
	util.SetLabelsForResource(service, util.LabelInput{
		Name:            forwarderName,
		Component:       util.ComponentBackend,
		ComponentLabels: cr.Spec.Fluentbit.Labels,
	}, nil)
	return service
}

// forwarderStorage returns the volumes and mounts for node logs, FluentBit state and
// buffers, the configuration, and temporary files.
func forwarderStorage(cr *loggingService.LoggingService) ([]corev1.Volume, []corev1.VolumeMount) {
	spec := cr.Spec.Fluentbit
	var volumes []corev1.Volume
	var mounts []corev1.VolumeMount

	if cr.Spec.ContainerRuntimeType == "docker" {
		if cr.Spec.OSKind != "ubuntu" {
			volumes = append(volumes, hostPathVolume("sysconfigdocker", "/etc/sysconfig/docker", corev1.HostPathFileOrCreate))
			mounts = append(mounts, corev1.VolumeMount{Name: "sysconfigdocker", MountPath: "/etc/sysconfig/docker", ReadOnly: true})
		}
		volumes = append(volumes,
			hostPathVolume("dockerdaemon", "/etc/docker/daemon.json", corev1.HostPathFileOrCreate),
			hostPathVolume("dockerhostname", "/etc/hostname", corev1.HostPathFileOrCreate),
			hostPathVolume("varlibdockercontainers", "/var/lib/docker/containers", corev1.HostPathUnset),
		)
		mounts = append(mounts,
			corev1.VolumeMount{Name: "dockerdaemon", MountPath: "/etc/docker/daemon.json", ReadOnly: true},
			corev1.VolumeMount{Name: "dockerhostname", MountPath: "/etc/docker-hostname", ReadOnly: true},
			corev1.VolumeMount{Name: "varlibdockercontainers", MountPath: "/var/lib/docker/containers", ReadOnly: true},
		)
	}

	volumes = append(volumes, hostPathVolume("varlog", "/var/log", corev1.HostPathUnset))
	mounts = append(mounts, corev1.VolumeMount{Name: "varlog", MountPath: "/var/log", ReadOnly: true})

	// The read offsets survive a pod restart only with a persistent storage profile.
	if spec.UsesPersistentOffsets() {
		volumes = append(volumes, hostPathVolume("fluentbit-state", "/var/lib/fluent-bit/state", corev1.HostPathDirectoryOrCreate))
	} else {
		size := resource.MustParse(spec.EffectiveMemoryOnlyStateSizeLimit())
		volumes = append(volumes, corev1.Volume{
			Name: "fluentbit-state",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium:    corev1.StorageMediumMemory,
				SizeLimit: &size,
			}},
		})
	}
	mounts = append(mounts, corev1.VolumeMount{Name: "fluentbit-state", MountPath: "/fluent-bit/state"})

	if spec.UsesFilesystemBuffer() {
		volumes = append(volumes, hostPathVolume("fluentbit-storage", "/var/lib/fluent-bit/storage", corev1.HostPathDirectoryOrCreate))
		mounts = append(mounts, corev1.VolumeMount{Name: "fluentbit-storage", MountPath: "/fluent-bit/storage"})
	}

	volumes = append(volumes,
		corev1.Volume{
			Name: configVolume,
			VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: forwarderName},
				DefaultMode:          ptr.To(secretMode),
			}},
		},
		emptyDirVolume(tmpVolume, tmpVolumeSize),
	)
	mounts = append(mounts,
		corev1.VolumeMount{Name: configVolume, MountPath: configDir, ReadOnly: true},
		corev1.VolumeMount{Name: tmpVolume, MountPath: "/tmp"},
	)
	return volumes, mounts
}

// forwarderPodSecurityContext adds the spc_t SELinux type on OpenShift, where node logs
// and persistent state carry host SELinux labels.
func forwarderPodSecurityContext(openshift bool) *corev1.PodSecurityContext {
	sc := &corev1.PodSecurityContext{
		SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
	}
	if openshift {
		sc.SELinuxOptions = &corev1.SELinuxOptions{Type: "spc_t"}
	}
	return sc
}

// forwarderSecurityContext runs the forwarder as root because node log directories are
// root-owned with mode 0750. Without the privileged mode, the container keeps only
// DAC_OVERRIDE.
func forwarderSecurityContext(privileged bool) *corev1.SecurityContext {
	sc := &corev1.SecurityContext{
		Privileged:             ptr.To(privileged),
		ReadOnlyRootFilesystem: ptr.To(true),
		RunAsNonRoot:           ptr.To(false),
		RunAsUser:              ptr.To(int64(0)),
		RunAsGroup:             ptr.To(int64(0)),
	}
	if !privileged {
		sc.AllowPrivilegeEscalation = ptr.To(false)
		sc.Capabilities = &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
			Add:  []corev1.Capability{"DAC_OVERRIDE"},
		}
	}
	return sc
}
