package fluentbit_forwarder_aggregator

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
)

func TestForwarderDaemonSetStorageProfiles(t *testing.T) {
	tests := []struct {
		name              string
		profile           string
		persistentStorage bool
	}{
		{name: "memory only", profile: loggingService.FluentbitStorageProfileMemoryOnly},
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
					DockerImage:     "fluent-bit:test",
					ConfigmapReload: &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
					StorageProfile:  test.profile,
				},
			}}

			daemonSet, err := forwarderDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
			if err != nil {
				t.Fatalf("render Fluent Bit forwarder DaemonSet: %v", err)
			}
			assertForwarderStorageProfile(t, daemonSet.Spec.Template.Spec, test.persistentStorage)
		})
	}
}

func assertForwarderStorageProfile(t *testing.T, podSpec corev1.PodSpec, persistentStorage bool) {
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
	} else if state == nil || state.EmptyDir == nil || state.EmptyDir.Medium != corev1.StorageMediumMemory {
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
