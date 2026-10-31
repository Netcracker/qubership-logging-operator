package forwarderaggregator

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestForwarderDaemonSetHardeningExceptions(t *testing.T) {
	cr := &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				DockerImage:     "fluent-bit:test",
				ConfigmapReload: &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
			},
		},
	}

	cr.Spec.ContainerRuntimeType = "containerd"
	daemonSet := ForwarderDaemonSet(cr, defaults.Get())

	podSpec := daemonSet.Spec.Template.Spec
	assertForwarderPodContext(t, podSpec.SecurityContext)
	assertForwarderContainerSecurity(t, podSpec.Containers[1])

	cr.Spec.OpenshiftDeploy = true
	openShiftDaemonSet := ForwarderDaemonSet(cr, defaults.Get())
	assertOpenShiftForwarderSecurity(t, openShiftDaemonSet.Spec.Template.Spec)
}

func assertForwarderPodContext(t *testing.T, context *corev1.PodSecurityContext) {
	t.Helper()
	if context == nil || context.SeccompProfile == nil ||
		context.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("pod must use the RuntimeDefault seccomp profile")
	}
}

func assertForwarderContainerSecurity(t *testing.T, forwarder corev1.Container) {
	t.Helper()
	context := forwarder.SecurityContext
	if context == nil {
		t.Fatal("forwarder security context is missing")
	}
	if context.RunAsUser == nil || *context.RunAsUser != 0 || context.RunAsNonRoot == nil || *context.RunAsNonRoot ||
		context.RunAsGroup == nil || *context.RunAsGroup != 0 {
		t.Error("forwarder must run as root to access node logs and its existing state under /var/log")
	}
	if context.ReadOnlyRootFilesystem == nil || !*context.ReadOnlyRootFilesystem {
		t.Error("forwarder must use a read-only root filesystem")
	}
	if !hasCapability(context.Capabilities.Add, corev1.Capability("DAC_OVERRIDE")) {
		t.Error("forwarder must add DAC_OVERRIDE to write its existing state on the /var/log hostPath")
	}
	if !hasReadOnlyMount(forwarder.VolumeMounts, "varlog", "/var/log") {
		t.Error("forwarder must mount node logs read-only")
	}
	if !hasMount(forwarder.VolumeMounts, "tmp", "/tmp") {
		t.Error("forwarder must mount the tmp volume at /tmp")
	}
}

func assertOpenShiftForwarderSecurity(t *testing.T, podSpec corev1.PodSpec) {
	t.Helper()
	if podSpec.SecurityContext == nil || podSpec.SecurityContext.SELinuxOptions == nil ||
		podSpec.SecurityContext.SELinuxOptions.Type != "spc_t" {
		t.Error("OpenShift forwarder pod must use spc_t to access var_log_t host paths")
	}
	for _, container := range podSpec.Containers {
		expectedGroup := int64(1001)
		if container.Name == "logging-fluentbit-forwarder" {
			expectedGroup = 0
		}
		if container.SecurityContext == nil || container.SecurityContext.RunAsGroup == nil ||
			*container.SecurityContext.RunAsGroup != expectedGroup {
			t.Errorf("OpenShift container %s must use GID %d", container.Name, expectedGroup)
		}
	}
}

func TestAggregatorStatefulSetSecurityContext(t *testing.T) {
	for _, openshift := range []bool{false, true} {
		t.Run(map[bool]string{false: "kubernetes", true: "openshift"}[openshift], func(t *testing.T) {
			statefulSet := AggregatorStatefulSet(newAggregatorSecurityLoggingService(openshift), defaults.Get())

			podSpec := statefulSet.Spec.Template.Spec
			assertAggregatorPodContext(t, podSpec.SecurityContext)
			assertAggregatorContainers(t, podSpec.Containers)
		})
	}
}

func TestAggregatorStatefulSetStorageSizeLimit(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		want       string
	}{
		{name: "default", want: loggingService.FluentbitAggregatorDefaultStorageSizeLimit},
		{name: "configured", configured: "6Gi", want: "6Gi"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cr := newAggregatorSecurityLoggingService(false)
			cr.Spec.Fluentbit.Aggregator.StorageSizeLimit = test.configured
			statefulSet := AggregatorStatefulSet(cr, defaults.Get())

			var storage *corev1.Volume
			for index := range statefulSet.Spec.Template.Spec.Volumes {
				volume := &statefulSet.Spec.Template.Spec.Volumes[index]
				if volume.Name == "storage" {
					storage = volume
					break
				}
			}
			if storage == nil || storage.EmptyDir == nil || storage.EmptyDir.SizeLimit == nil ||
				storage.EmptyDir.SizeLimit.String() != test.want {
				t.Errorf("unexpected aggregator storage volume: %#v", storage)
			}
		})
	}
}

func newAggregatorSecurityLoggingService(openshift bool) *loggingService.LoggingService {
	resources := &corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
	}
	return &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
		OpenshiftDeploy: openshift,
		Fluentbit: &loggingService.Fluentbit{
			DockerImage: "fluent-bit:test",
			Aggregator: &loggingService.FluentbitAggregator{
				DockerImage: "fluent-bit:test",
				Resources:   resources,
				ConfigmapReload: &loggingService.ConfigmapReload{
					DockerImage: "configmap-reload:test",
				},
			},
		},
	}}
}

func assertAggregatorPodContext(t *testing.T, context *corev1.PodSecurityContext) {
	t.Helper()
	if context == nil {
		t.Fatal("aggregator pod security context is missing")
	}
	if context.RunAsNonRoot == nil || !*context.RunAsNonRoot || context.SeccompProfile == nil ||
		context.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("aggregator pod must run as non-root with the RuntimeDefault seccomp profile")
	}
	if context.RunAsUser == nil || *context.RunAsUser != 1001 {
		t.Error("aggregator pod must use UID 1001")
	}
	if context.RunAsGroup == nil || *context.RunAsGroup != 1001 {
		t.Error("aggregator pod must use GID 1001")
	}
}

func assertAggregatorContainers(t *testing.T, containers []corev1.Container) {
	t.Helper()
	for _, container := range containers {
		assertAggregatorContainerContext(t, container)
	}
	if !hasWritableMount(containers[1].VolumeMounts, "storage", "/fluent-bit/storage") {
		t.Error("aggregator must retain its writable storage mount")
	}
}

func assertAggregatorContainerContext(t *testing.T, container corev1.Container) {
	t.Helper()
	context := container.SecurityContext
	if context == nil {
		t.Fatalf("container %s has no security context", container.Name)
	}
	if context.RunAsNonRoot == nil || !*context.RunAsNonRoot || context.AllowPrivilegeEscalation == nil ||
		*context.AllowPrivilegeEscalation || context.ReadOnlyRootFilesystem == nil ||
		!*context.ReadOnlyRootFilesystem || context.Capabilities == nil ||
		!hasCapability(context.Capabilities.Drop, corev1.Capability("ALL")) {
		t.Errorf("container %s must use the hardened non-root security context", container.Name)
	}
	if !hasMount(container.VolumeMounts, "tmp", "/tmp") {
		t.Errorf("container %s must mount the tmp volume", container.Name)
	}
	if context.RunAsUser == nil || *context.RunAsUser != 1001 {
		t.Errorf("container %s must use UID 1001", container.Name)
	}
	if context.RunAsGroup == nil || *context.RunAsGroup != 1001 {
		t.Errorf("container %s must use GID 1001", container.Name)
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

func hasWritableMount(mounts []corev1.VolumeMount, name, path string) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path && !mount.ReadOnly {
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
