package fluentbit_forwarder_aggregator

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
)

func TestForwarderDaemonSetStorageProfiles(t *testing.T) {
	tests := []struct {
		name                   string
		profile                string
		memoryStateSizeLimit   string
		expectedStateSizeLimit string
		persistentStorage      bool
	}{
		{
			name:                   "memory only with default limit",
			profile:                loggingService.FluentbitStorageProfileMemoryOnly,
			expectedStateSizeLimit: loggingService.FluentbitDefaultMemoryOnlyStateSizeLimit,
		},
		{
			name:                   "memory only with custom limit",
			profile:                loggingService.FluentbitStorageProfileMemoryOnly,
			memoryStateSizeLimit:   "96Mi",
			expectedStateSizeLimit: "96Mi",
		},
		{
			name:              "node persistent",
			profile:           loggingService.FluentbitStorageProfileNodePersistent,
			persistentStorage: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
				Fluentbit: &loggingService.Fluentbit{
					DockerImage:              "fluent-bit:test",
					ConfigmapReload:          &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
					StorageProfile:           test.profile,
					MemoryOnlyStateSizeLimit: test.memoryStateSizeLimit,
				},
			}}

			daemonSet, err := forwarderDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
			if err != nil {
				t.Fatalf("render Fluent Bit forwarder DaemonSet: %v", err)
			}
			assertForwarderStorageProfile(t, daemonSet.Spec.Template.Spec, test.expectedStateSizeLimit,
				test.persistentStorage)
		})
	}
}

func TestForwarderDaemonSetSecurityContext(t *testing.T) {
	cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
		Fluentbit: &loggingService.Fluentbit{
			DockerImage:     "fluent-bit:test",
			ConfigmapReload: &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
		},
	}}

	daemonSet, err := forwarderDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
	if err != nil {
		t.Fatalf("render Fluent Bit forwarder DaemonSet: %v", err)
	}
	assertForwarderPodSecurity(t, daemonSet.Spec.Template.Spec)

	cr.Spec.OpenshiftDeploy = true
	openShiftDaemonSet, err := forwarderDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
	if err != nil {
		t.Fatalf("render OpenShift Fluent Bit forwarder DaemonSet: %v", err)
	}
	context := openShiftDaemonSet.Spec.Template.Spec.SecurityContext
	if context == nil || context.SELinuxOptions == nil || context.SELinuxOptions.Type != "spc_t" {
		t.Error("OpenShift forwarder pod must use spc_t to access host logs and persistent storage")
	}
}

func assertForwarderPodSecurity(t *testing.T, podSpec corev1.PodSpec) {
	t.Helper()
	if podSpec.SecurityContext == nil || podSpec.SecurityContext.SeccompProfile == nil ||
		podSpec.SecurityContext.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("pod must use the RuntimeDefault seccomp profile")
	}

	reloader := podSpec.Containers[0]
	if reloader.SecurityContext == nil || reloader.SecurityContext.RunAsNonRoot == nil ||
		!*reloader.SecurityContext.RunAsNonRoot || reloader.SecurityContext.RunAsUser == nil ||
		*reloader.SecurityContext.RunAsUser != 1001 || reloader.SecurityContext.ReadOnlyRootFilesystem == nil ||
		!*reloader.SecurityContext.ReadOnlyRootFilesystem {
		t.Error("configmap-reload must run as UID 1001 with a read-only root filesystem")
	}

	forwarder := podSpec.Containers[1]
	context := forwarder.SecurityContext
	if context == nil || context.RunAsUser == nil || *context.RunAsUser != 0 || context.RunAsGroup == nil ||
		*context.RunAsGroup != 0 || context.RunAsNonRoot == nil || *context.RunAsNonRoot ||
		context.ReadOnlyRootFilesystem == nil || !*context.ReadOnlyRootFilesystem {
		t.Error("forwarder must run as root with a read-only root filesystem")
	}
	if !forwarderHasCapability(context.Capabilities.Add, corev1.Capability("DAC_OVERRIDE")) {
		t.Error("unprivileged forwarder must add DAC_OVERRIDE to read root-owned node logs")
	}
	if !forwarderStorageProfileHasMount(reloader.VolumeMounts, "tmp", "/tmp", false) ||
		!forwarderStorageProfileHasMount(forwarder.VolumeMounts, "tmp", "/tmp", false) {
		t.Error("Fluent Bit forwarder containers must mount writable temporary storage")
	}
	tmp := forwarderStorageProfileFindVolume(podSpec.Volumes, "tmp")
	if tmp == nil || tmp.EmptyDir == nil || tmp.EmptyDir.SizeLimit == nil || tmp.EmptyDir.SizeLimit.String() != "100Mi" {
		t.Errorf("unexpected temporary storage volume: %#v", tmp)
	}
}

func forwarderHasCapability(capabilities []corev1.Capability, expected corev1.Capability) bool {
	for _, capability := range capabilities {
		if capability == expected {
			return true
		}
	}
	return false
}

func assertForwarderStorageProfile(t *testing.T, podSpec corev1.PodSpec, expectedStateSizeLimit string,
	persistentStorage bool) {
	t.Helper()
	forwarder := podSpec.Containers[1]
	if !forwarderStorageProfileHasMount(forwarder.VolumeMounts, "varlog", "/var/log", true) {
		t.Error("forwarder must mount node logs read-only")
	}
	if !forwarderStorageProfileHasMount(forwarder.VolumeMounts, "fluentbit-state", "/fluent-bit/state", false) {
		t.Error("forwarder must mount writable state storage")
	}

	state := forwarderStorageProfileFindVolume(podSpec.Volumes, "fluentbit-state")
	if persistentStorage {
		if state == nil || state.HostPath == nil || state.HostPath.Path != "/var/lib/fluent-bit/state" {
			t.Errorf("unexpected persistent state volume: %#v", state)
		}
		storage := forwarderStorageProfileFindVolume(podSpec.Volumes, "fluentbit-storage")
		if storage == nil || storage.HostPath == nil || storage.HostPath.Path != "/var/lib/fluent-bit/storage" ||
			!forwarderStorageProfileHasMount(forwarder.VolumeMounts, "fluentbit-storage", "/fluent-bit/storage", false) {
			t.Errorf("persistent input buffer is not configured correctly: %#v", storage)
		}
	} else if state == nil || state.EmptyDir == nil || state.EmptyDir.Medium != corev1.StorageMediumMemory ||
		state.EmptyDir.SizeLimit == nil || state.EmptyDir.SizeLimit.String() != expectedStateSizeLimit {
		t.Errorf("unexpected memory state volume: %#v", state)
	}
}

func forwarderStorageProfileHasMount(mounts []corev1.VolumeMount, name, path string, readOnly bool) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path && mount.ReadOnly == readOnly {
			return true
		}
	}
	return false
}

func forwarderStorageProfileFindVolume(volumes []corev1.Volume, name string) *corev1.Volume {
	for index := range volumes {
		if volumes[index].Name == name {
			return &volumes[index]
		}
	}
	return nil
}
