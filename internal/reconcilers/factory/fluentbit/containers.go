package fluentbit

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build"
	corev1 "k8s.io/api/core/v1"
)

// buildConfigmapReloadContainer constructs the configmap-reload sidecar that watches
// /fluent-bit/etc for changes and POSTs to fluent-bit's reload endpoint. Image and
// resources come from cr.Spec.Fluentbit.ConfigmapReload with code-level fallbacks.
func buildConfigmapReloadContainer(spec *loggingService.Fluentbit, def config.FluentbitDefaults) corev1.Container {
	image := ""
	resources := def.ConfigmapReloadResources.DeepCopy()
	if spec.ConfigmapReload != nil {
		image = spec.ConfigmapReload.DockerImage
		if spec.ConfigmapReload.Resources != nil {
			resources = spec.ConfigmapReload.Resources.DeepCopy()
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

// buildMainContainer constructs the fluent-bit container with all conditional env
// vars, volume mounts, and resources.
func buildMainContainer(cr *loggingService.LoggingService, def config.FluentbitDefaults) corev1.Container {
	spec := cr.Spec.Fluentbit
	resources := def.Resources.DeepCopy()
	if spec.Resources != nil {
		resources = spec.Resources.DeepCopy()
	}
	c := build.NewContainer(MainContainer, build.ContainerOpts{
		Image:           spec.DockerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             buildMainEnv(spec),
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: def.HTTPPort, Protocol: corev1.ProtocolTCP},
			{Name: "log-to-metrics", ContainerPort: def.LogToMetricsPort, Protocol: corev1.ProtocolTCP},
		},
		LivenessProbe:   def.LivenessProbe.DeepCopy(),
		ReadinessProbe:  def.ReadinessProbe.DeepCopy(),
		Resources:       *resources,
		VolumeMounts:    buildMainVolumeMounts(cr),
		SecurityContext: &corev1.SecurityContext{Privileged: boolPtr(spec.SecurityContextPrivileged)},
	})

	if len(spec.AdditionalVolumeMounts) > 0 {
		c.VolumeMounts = append(c.VolumeMounts, spec.AdditionalVolumeMounts...)
	}
	return c
}

func boolPtr(b bool) *bool { return &b }
