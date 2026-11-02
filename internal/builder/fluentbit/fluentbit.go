// Package fluentbit builds the FluentBit DaemonSet and Service of the standard
// (non-HA) mode from the LoggingService custom resource.
package fluentbit

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

const (
	name            = "logging-fluentbit"
	reloadContainer = "configmap-reload"
	technology      = "c"
	configVolume    = "config"
	tmpVolume       = "tmp"
	stateVolume     = "fluentbit-state"
	storageVolume   = "fluentbit-storage"
	secretFileMode  = int32(420)
)

// DaemonSet returns the FluentBit DaemonSet. The caller checks that cr.Spec.Fluentbit
// is set and sets cr.Spec.ContainerRuntimeType to the detected runtime.
func DaemonSet(cr *loggingService.LoggingService, cfg *defaults.Defaults) *appsv1.DaemonSet {
	spec := cr.Spec.Fluentbit
	def := cfg.Fluentbit

	reloader := reloaderContainer(spec, def)
	collector := collectorContainer(cr, def)
	collector.VolumeMounts = append(collector.VolumeMounts, spec.AdditionalVolumeMounts...)

	tolerations := def.Tolerations
	if spec.Tolerations != nil {
		tolerations = spec.Tolerations
	}
	podSpec := corev1.PodSpec{
		TerminationGracePeriodSeconds: ptr.To(def.TerminationGracePeriod),
		ServiceAccountName:            name,
		SecurityContext:               podSecurityContext(cr.Spec.OpenshiftDeploy),
		Tolerations:                   tolerations,
		Containers:                    []corev1.Container{reloader, collector},
		Volumes:                       append(volumes(cr, def), spec.AdditionalVolumes...),
		NodeSelector:                  builder.NodeSelector(spec.NodeSelectorKey, spec.NodeSelectorValue),
		Affinity:                      spec.Affinity,
	}
	if strings.TrimSpace(spec.PriorityClassName) != "" {
		podSpec.PriorityClassName = spec.PriorityClassName
	}

	maxUnavailable := def.MaxUnavailable
	daemonSet := builder.NewDaemonSet(name, cr.GetNamespace(), builder.DaemonSetOpts{
		Selector:        selector(cr.Spec.OpenshiftDeploy),
		MinReadySeconds: def.MinReadySeconds,
		UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
			Type:          appsv1.RollingUpdateDaemonSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{MaxUnavailable: &maxUnavailable},
		},
		PodSpec:        podSpec,
		PodLabels:      workloadLabels(cr.Spec.OpenshiftDeploy),
		PodAnnotations: spec.Annotations,
		Labels:         workloadLabels(cr.Spec.OpenshiftDeploy),
		Annotations:    spec.Annotations,
	})
	daemonSet.Spec.Template.Name = name
	util.SetLabelsForWorkload(daemonSet, &daemonSet.Spec.Template.Labels, util.LabelInput{
		Name:            name,
		Component:       util.ComponentBackend,
		Instance:        util.GetInstanceLabel(name, cr.GetNamespace()),
		Version:         util.GetTagFromImage(spec.DockerImage),
		Technology:      technology,
		ComponentLabels: spec.Labels,
	})
	return daemonSet
}

// Service returns the FluentBit Service. The caller checks that cr.Spec.Fluentbit is set.
func Service(cr *loggingService.LoggingService, cfg *defaults.Defaults) *corev1.Service {
	def := cfg.Fluentbit
	service := builder.NewService(name, cr.GetNamespace(), builder.ServiceOpts{
		Labels:   map[string]string{"component": name, "provider": provider(cr.Spec.OpenshiftDeploy)},
		Selector: map[string]string{"name": name},
		Ports: []corev1.ServicePort{
			servicePort(name, def.HTTPPort),
			servicePort("log-to-metrics", def.LogToMetricsPort),
		},
	})
	util.SetLabelsForResource(service, util.LabelInput{
		Name:            name,
		Component:       util.ComponentBackend,
		ComponentLabels: cr.Spec.Fluentbit.Labels,
	}, nil)
	return service
}

func provider(openshift bool) string {
	if openshift {
		return "openshift"
	}
	return "kubernetes"
}

func selector(openshift bool) map[string]string {
	return map[string]string{"component": name, "provider": provider(openshift)}
}

// workloadLabels are the initial labels of the DaemonSet and its pods. The
// app.kubernetes.io/technology value is replaced by util.SetLabelsForWorkload.
func workloadLabels(openshift bool) map[string]string {
	labels := selector(openshift)
	labels["app.kubernetes.io/technology"] = "cpp"
	labels["kubernetes.io/cluster-service"] = "true"
	return labels
}

func servicePort(portName string, port int32) corev1.ServicePort {
	return corev1.ServicePort{Name: portName, Port: port, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt32(port)}
}

func reloaderContainer(spec *loggingService.Fluentbit, def defaults.FluentbitDefaults) corev1.Container {
	image := ""
	resources := def.ConfigmapReloadResources.DeepCopy()
	if spec.ConfigmapReload != nil {
		image = spec.ConfigmapReload.DockerImage
		if spec.ConfigmapReload.Resources != nil {
			resources = spec.ConfigmapReload.Resources.DeepCopy()
		}
	}
	return builder.NewContainer(reloadContainer, builder.ContainerOpts{
		Image: image,
		Args:  def.ConfigmapReloadArgs,
		VolumeMounts: []corev1.VolumeMount{
			{Name: configVolume, MountPath: "/fluent-bit/etc", ReadOnly: true},
			{Name: tmpVolume, MountPath: "/tmp"},
		},
		Resources: *resources,
		Ports:     []corev1.ContainerPort{{Name: "reload-metrics", ContainerPort: def.ConfigmapReloadPort}},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			ReadOnlyRootFilesystem:   ptr.To(true),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			RunAsNonRoot:             ptr.To(true),
			RunAsUser:                ptr.To(def.ReloaderID),
			RunAsGroup:               ptr.To(def.ReloaderID),
		},
	})
}

func collectorContainer(cr *loggingService.LoggingService, def defaults.FluentbitDefaults) corev1.Container {
	spec := cr.Spec.Fluentbit
	resources := def.Resources.DeepCopy()
	if spec.Resources != nil {
		resources = spec.Resources.DeepCopy()
	}
	return builder.NewContainer(name, builder.ContainerOpts{
		Image:           spec.DockerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: collectorSecurityContext(spec.SecurityContextPrivileged),
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: def.HTTPPort, Protocol: corev1.ProtocolTCP},
			{Name: "log-to-metrics", ContainerPort: def.LogToMetricsPort, Protocol: corev1.ProtocolTCP},
		},
		LivenessProbe:  def.LivenessProbe.DeepCopy(),
		ReadinessProbe: def.ReadinessProbe.DeepCopy(),
		Resources:      *resources,
		Env: []corev1.EnvVar{{
			Name:      "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}},
		}},
		VolumeMounts: volumeMounts(cr),
	})
}

// collectorSecurityContext runs the collector as root because node log directories
// are root-owned with mode 0750. An unprivileged collector keeps only DAC_OVERRIDE.
func collectorSecurityContext(privileged bool) *corev1.SecurityContext {
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

// podSecurityContext adds the spc_t SELinux type on OpenShift, where host SELinux
// labels protect node logs and the persistent state directories.
func podSecurityContext(openshift bool) *corev1.PodSecurityContext {
	sc := &corev1.PodSecurityContext{
		SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
	}
	if openshift {
		sc.SELinuxOptions = &corev1.SELinuxOptions{Type: "spc_t"}
	}
	return sc
}

// stateSizeLimit falls back to the default limit when the configured value is not a
// valid quantity.
func stateSizeLimit(spec *loggingService.Fluentbit) resource.Quantity {
	if quantity, err := resource.ParseQuantity(spec.EffectiveMemoryOnlyStateSizeLimit()); err == nil {
		return quantity
	}
	return resource.MustParse(loggingService.FluentbitDefaultMemoryOnlyStateSizeLimit)
}
