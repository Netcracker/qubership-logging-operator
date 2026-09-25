package fluentbit

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
)

func TestFluentbitDaemonSetStorageProfiles(t *testing.T) {
	tests := []struct {
		name                   string
		profile                string
		memoryStateSizeLimit   string
		expectedStateSizeLimit string
		persistentOffsets      bool
		persistentInputBuffer  bool
	}{
		{
			name:                   "memory only with default limit",
			profile:                loggingService.FluentbitStorageProfileMemoryOnly,
			expectedStateSizeLimit: loggingService.FluentbitDefaultMemoryOnlyStateSizeLimit,
		},
		{
			name:                   "memory only with custom limit",
			profile:                loggingService.FluentbitStorageProfileMemoryOnly,
			memoryStateSizeLimit:   "128Mi",
			expectedStateSizeLimit: "128Mi",
		},
		{
			name:              "persistent offsets",
			profile:           loggingService.FluentbitStorageProfilePersistentOffsets,
			persistentOffsets: true,
		},
		{
			name:                  "node persistent",
			profile:               loggingService.FluentbitStorageProfileNodePersistent,
			persistentOffsets:     true,
			persistentInputBuffer: true,
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

			daemonSet, err := fluentbitDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
			if err != nil {
				t.Fatalf("render Fluent Bit DaemonSet: %v", err)
			}
			assertFluentbitStorageProfile(t, daemonSet.Spec.Template.Spec, test.expectedStateSizeLimit,
				test.persistentOffsets, test.persistentInputBuffer)
		})
	}
}

func TestFluentbitDaemonSetSecurityContext(t *testing.T) {
	for _, privileged := range []bool{false, true} {
		t.Run(map[bool]string{false: "unprivileged", true: "privileged"}[privileged], func(t *testing.T) {
			cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
				Fluentbit: &loggingService.Fluentbit{
					DockerImage:               "fluent-bit:test",
					ConfigmapReload:           &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
					SecurityContextPrivileged: privileged,
				},
			}}

			daemonSet, err := fluentbitDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
			if err != nil {
				t.Fatalf("render Fluent Bit DaemonSet: %v", err)
			}

			assertFluentbitPodSecurity(t, daemonSet.Spec.Template.Spec, privileged)
		})
	}
}

func TestFluentbitOpenShiftSELinuxContext(t *testing.T) {
	cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
		OpenshiftDeploy: true,
		Fluentbit: &loggingService.Fluentbit{
			DockerImage:     "fluent-bit:test",
			ConfigmapReload: &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
		},
	}}

	daemonSet, err := fluentbitDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
	if err != nil {
		t.Fatalf("render OpenShift Fluent Bit DaemonSet: %v", err)
	}

	context := daemonSet.Spec.Template.Spec.SecurityContext
	if context == nil || context.SELinuxOptions == nil || context.SELinuxOptions.Type != "spc_t" {
		t.Error("OpenShift collector pod must use spc_t to access host logs and persistent storage")
	}
}

func assertFluentbitPodSecurity(t *testing.T, podSpec corev1.PodSpec, privileged bool) {
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

	collector := podSpec.Containers[1]
	context := collector.SecurityContext
	if context == nil || context.RunAsUser == nil || *context.RunAsUser != 0 || context.RunAsGroup == nil ||
		*context.RunAsGroup != 0 || context.RunAsNonRoot == nil || *context.RunAsNonRoot ||
		context.ReadOnlyRootFilesystem == nil || !*context.ReadOnlyRootFilesystem {
		t.Error("collector must run as root with a read-only root filesystem")
	}
	if context.Privileged == nil || *context.Privileged != privileged {
		t.Errorf("collector privileged setting = %v, want %v", context.Privileged, privileged)
	}
	if !privileged && !fluentbitHasCapability(context.Capabilities.Add, corev1.Capability("DAC_OVERRIDE")) {
		t.Error("unprivileged collector must add DAC_OVERRIDE to read root-owned node logs")
	}
	if !storageProfileHasMount(reloader.VolumeMounts, "tmp", "/tmp", false) ||
		!storageProfileHasMount(collector.VolumeMounts, "tmp", "/tmp", false) {
		t.Error("Fluent Bit containers must mount writable temporary storage")
	}
	tmp := storageProfileFindVolume(podSpec.Volumes, "tmp")
	if tmp == nil || tmp.EmptyDir == nil || tmp.EmptyDir.SizeLimit == nil || tmp.EmptyDir.SizeLimit.String() != "100Mi" {
		t.Errorf("unexpected temporary storage volume: %#v", tmp)
	}
}

func fluentbitHasCapability(capabilities []corev1.Capability, expected corev1.Capability) bool {
	for _, capability := range capabilities {
		if capability == expected {
			return true
		}
	}
	return false
}

func assertFluentbitStorageProfile(t *testing.T, podSpec corev1.PodSpec, expectedStateSizeLimit string,
	persistentOffsets, persistentInputBuffer bool) {
	t.Helper()
	collector := podSpec.Containers[1]
	if !storageProfileHasMount(collector.VolumeMounts, "varlog", "/var/log", true) {
		t.Error("collector must mount node logs read-only")
	}
	if !storageProfileHasMount(collector.VolumeMounts, "fluentbit-state", "/fluent-bit/state", false) {
		t.Error("collector must mount writable state storage")
	}

	state := storageProfileFindVolume(podSpec.Volumes, "fluentbit-state")
	if state == nil {
		t.Fatal("state volume is missing")
	}
	if persistentOffsets {
		if state.HostPath == nil || state.HostPath.Path != "/var/lib/fluent-bit/state" ||
			state.HostPath.Type == nil || *state.HostPath.Type != corev1.HostPathDirectoryOrCreate {
			t.Errorf("unexpected persistent state volume: %#v", state.VolumeSource)
		}
	} else if state.EmptyDir == nil || state.EmptyDir.Medium != corev1.StorageMediumMemory ||
		state.EmptyDir.SizeLimit == nil || state.EmptyDir.SizeLimit.String() != expectedStateSizeLimit {
		t.Errorf("unexpected memory state volume: %#v", state.VolumeSource)
	}

	storage := storageProfileFindVolume(podSpec.Volumes, "fluentbit-storage")
	if persistentInputBuffer {
		if storage == nil || storage.HostPath == nil || storage.HostPath.Path != "/var/lib/fluent-bit/storage" ||
			!storageProfileHasMount(collector.VolumeMounts, "fluentbit-storage", "/fluent-bit/storage", false) {
			t.Errorf("persistent input buffer is not configured correctly: %#v", storage)
		}
	} else if storage != nil {
		t.Error("memory buffering profile must not create node buffer storage")
	}
}

func storageProfileHasMount(mounts []corev1.VolumeMount, name, path string, readOnly bool) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path && mount.ReadOnly == readOnly {
			return true
		}
	}
	return false
}

func storageProfileFindVolume(volumes []corev1.Volume, name string) *corev1.Volume {
	for index := range volumes {
		if volumes[index].Name == name {
			return &volumes[index]
		}
	}
	return nil
}
