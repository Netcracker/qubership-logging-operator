package fluentd

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
)

func TestFluentdDaemonSetSecurityContext(t *testing.T) {
	for _, privileged := range []bool{false, true} {
		t.Run(map[bool]string{false: "unprivileged", true: "privileged"}[privileged], func(t *testing.T) {
			cr := &loggingService.LoggingService{
				Spec: loggingService.LoggingServiceSpec{
					Fluentd: &loggingService.Fluentd{
						DockerImage:               "fluentd:test",
						SecurityContextPrivileged: privileged,
						ConfigmapReload: &loggingService.ConfigmapReload{
							DockerImage: "configmap-reload:test",
						},
					},
				},
			}

			cr.Spec.ContainerRuntimeType = "containerd"
			daemonSet := DaemonSet(cr, defaults.Get())

			podSpec := daemonSet.Spec.Template.Spec
			assertFluentdPodContext(t, podSpec.SecurityContext)
			assertFluentdReloaderContext(t, podSpec.Containers[0].SecurityContext)
			assertFluentdContainerSecurity(t, podSpec.Containers[1], privileged)
			assertContainersMountTmp(t, podSpec.Containers)
		})
	}
}

func assertFluentdPodContext(t *testing.T, context *corev1.PodSecurityContext) {
	t.Helper()
	if context == nil || context.RunAsUser == nil || *context.RunAsUser != 0 || context.RunAsNonRoot == nil ||
		*context.RunAsNonRoot || context.RunAsGroup == nil || *context.RunAsGroup != 0 ||
		context.SeccompProfile == nil || context.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("Fluentd pod must run as root with the RuntimeDefault seccomp profile")
	}
}

func assertFluentdReloaderContext(t *testing.T, context *corev1.SecurityContext) {
	t.Helper()
	if context == nil || context.RunAsUser == nil || *context.RunAsUser != 1001 || context.RunAsNonRoot == nil ||
		!*context.RunAsNonRoot || context.RunAsGroup == nil || *context.RunAsGroup != 1001 || !isHardened(context) {
		t.Error("configmap-reload must use the hardened non-root security context")
	}
}

func assertFluentdContainerSecurity(t *testing.T, container corev1.Container, privileged bool) {
	t.Helper()
	context := container.SecurityContext
	if context == nil {
		t.Fatal("Fluentd security context is missing")
	}
	if context.RunAsUser == nil || *context.RunAsUser != 0 || context.RunAsNonRoot == nil || *context.RunAsNonRoot ||
		context.RunAsGroup == nil || *context.RunAsGroup != 0 || context.Privileged == nil ||
		*context.Privileged != privileged || context.ReadOnlyRootFilesystem == nil ||
		!*context.ReadOnlyRootFilesystem {
		t.Error("Fluentd must retain root access and a read-only root filesystem")
	}
	if !privileged && !isHardened(context) {
		t.Error("unprivileged Fluentd must disable escalation and drop all capabilities")
	}
	if !hasWritableMount(container.VolumeMounts, "varlog", "/var/log") {
		t.Error("Fluentd must mount node logs read-write to persist position files")
	}
}

func assertContainersMountTmp(t *testing.T, containers []corev1.Container) {
	t.Helper()
	for _, container := range containers {
		if !hasMount(container.VolumeMounts, "tmp", "/tmp") {
			t.Errorf("container %s must mount the tmp volume", container.Name)
		}
	}
}

func TestFluentdOpenShiftSecurityContext(t *testing.T) {
	cr := &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			OpenshiftDeploy: true,
			Fluentd: &loggingService.Fluentd{
				DockerImage: "fluentd:test",
				ConfigmapReload: &loggingService.ConfigmapReload{
					DockerImage: "configmap-reload:test",
				},
			},
		},
	}
	cr.Spec.ContainerRuntimeType = "containerd"
	daemonSet := DaemonSet(cr, defaults.Get())
	podContext := daemonSet.Spec.Template.Spec.SecurityContext
	if podContext == nil || podContext.SELinuxOptions == nil || podContext.SELinuxOptions.Type != "spc_t" {
		t.Error("OpenShift Fluentd pod must use spc_t to access var_log_t host paths")
	}
}

// TestFluentdPodsCarryNameLabel guards the label the PodMonitor "fluentd-pod-monitor"
// selects on (charts/qubership-logging-operator/templates/fluentd/podmonitor.yaml).
func TestFluentdPodsCarryNameLabel(t *testing.T) {
	cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
		Fluentd: &loggingService.Fluentd{
			ConfigmapReload: &loggingService.ConfigmapReload{},
		},
	}}

	cr.Spec.ContainerRuntimeType = "containerd"
	daemonSet := DaemonSet(cr, defaults.Get())

	if got := daemonSet.Spec.Template.Labels["app.kubernetes.io/name"]; got != util.FluentdComponentName {
		t.Errorf("expected the pod label app.kubernetes.io/name=%s, got %q", util.FluentdComponentName, got)
	}
}

func isHardened(context *corev1.SecurityContext) bool {
	return context.AllowPrivilegeEscalation != nil && !*context.AllowPrivilegeEscalation &&
		context.ReadOnlyRootFilesystem != nil && *context.ReadOnlyRootFilesystem &&
		context.Capabilities != nil && hasCapability(context.Capabilities.Drop, corev1.Capability("ALL"))
}

func hasCapability(capabilities []corev1.Capability, expected corev1.Capability) bool {
	for _, capability := range capabilities {
		if capability == expected {
			return true
		}
	}
	return false
}

func hasMount(mounts []corev1.VolumeMount, name, path string) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path {
			return true
		}
	}
	return false
}

func hasWritableMount(mounts []corev1.VolumeMount, name, path string) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path && !mount.ReadOnly {
			return true
		}
	}
	return false
}
