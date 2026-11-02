package fluentbit

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	corev1 "k8s.io/api/core/v1"
)

func TestFluentbitDaemonSetHardeningExceptions(t *testing.T) {
	for _, privileged := range []bool{false, true} {
		t.Run(map[bool]string{false: "unprivileged", true: "privileged"}[privileged], func(t *testing.T) {
			cr := &loggingService.LoggingService{
				Spec: loggingService.LoggingServiceSpec{
					Fluentbit: &loggingService.Fluentbit{
						DockerImage:               "fluent-bit:test",
						ConfigmapReload:           &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
						SecurityContextPrivileged: privileged,
					},
				},
			}

			daemonSet := DaemonSet(withContainerd(cr), defaults.Get())

			podSpec := daemonSet.Spec.Template.Spec
			assertRuntimeDefaultPodContext(t, podSpec.SecurityContext)
			assertFluentbitReloaderContext(t, podSpec.Containers[0].SecurityContext)
			assertFluentbitCollectorSecurity(t, podSpec.Containers[1], privileged)
		})
	}
}

func assertRuntimeDefaultPodContext(t *testing.T, context *corev1.PodSecurityContext) {
	t.Helper()
	if context == nil || context.SeccompProfile == nil ||
		context.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("pod must use the RuntimeDefault seccomp profile")
	}
}

func assertFluentbitReloaderContext(t *testing.T, context *corev1.SecurityContext) {
	t.Helper()
	if context == nil || context.ReadOnlyRootFilesystem == nil || !*context.ReadOnlyRootFilesystem ||
		context.RunAsNonRoot == nil || !*context.RunAsNonRoot || context.RunAsGroup == nil ||
		*context.RunAsGroup != 1001 {
		t.Error("configmap-reload must run as non-root with a read-only root filesystem")
	}
}

func assertFluentbitCollectorSecurity(t *testing.T, collector corev1.Container, privileged bool) {
	t.Helper()
	if collector.SecurityContext == nil {
		t.Fatal("collector security context is missing")
	}
	assertRootCollectorContext(t, collector.SecurityContext)
	if collector.SecurityContext.Privileged == nil || *collector.SecurityContext.Privileged != privileged {
		t.Errorf("collector privileged setting = %v, want %v", collector.SecurityContext.Privileged, privileged)
	}
	if !privileged && !hasCapability(collector.SecurityContext.Capabilities.Add, corev1.Capability("DAC_OVERRIDE")) {
		t.Error("unprivileged collector must add DAC_OVERRIDE to write its existing state on the /var/log hostPath")
	}
	if !hasReadOnlyMount(collector.VolumeMounts, "varlog", "/var/log") {
		t.Error("collector must mount node logs read-only")
	}
	if !hasMount(collector.VolumeMounts, "tmp", "/tmp") {
		t.Error("collector must mount the tmp volume at /tmp")
	}
}

func assertRootCollectorContext(t *testing.T, context *corev1.SecurityContext) {
	t.Helper()
	if context.RunAsUser == nil || *context.RunAsUser != 0 || context.RunAsNonRoot == nil ||
		*context.RunAsNonRoot || context.RunAsGroup == nil || *context.RunAsGroup != 0 {
		t.Error("collector must run as root to access node logs and its existing state under /var/log")
	}
	if context.ReadOnlyRootFilesystem == nil || !*context.ReadOnlyRootFilesystem {
		t.Error("collector must use a read-only root filesystem")
	}
}

func TestFluentbitOpenShiftSecurityContext(t *testing.T) {
	cr := &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			OpenshiftDeploy: true,
			Fluentbit: &loggingService.Fluentbit{
				DockerImage:     "fluent-bit:test",
				ConfigmapReload: &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
			},
		},
	}

	daemonSet := DaemonSet(withContainerd(cr), defaults.Get())
	podContext := daemonSet.Spec.Template.Spec.SecurityContext
	if podContext == nil || podContext.SELinuxOptions == nil || podContext.SELinuxOptions.Type != "spc_t" {
		t.Error("OpenShift collector pod must use spc_t to access var_log_t host paths")
	}
}

func hasCapability(capabilities []corev1.Capability, expected corev1.Capability) bool {
	for _, capability := range capabilities {
		if capability == expected {
			return true
		}
	}
	return false
}

func hasReadOnlyMount(mounts []corev1.VolumeMount, name, path string) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path && mount.ReadOnly {
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

func withContainerd(cr *loggingService.LoggingService) *loggingService.LoggingService {
	cr.Spec.ContainerRuntimeType = "containerd"
	return cr
}
