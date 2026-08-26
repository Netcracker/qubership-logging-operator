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
