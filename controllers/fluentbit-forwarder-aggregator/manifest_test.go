package fluentbit_forwarder_aggregator

import (
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
)

func TestForwarderConfigMapStorageProfiles(t *testing.T) {
	tests := []struct {
		name                  string
		profile               string
		wantStorageType       string
		wantFilesystemStorage bool
	}{
		{
			name:            "memory only",
			profile:         loggingService.FluentbitStorageProfileMemoryOnly,
			wantStorageType: "memory",
		},
		{
			name:                  "node persistent",
			profile:               loggingService.FluentbitStorageProfileNodePersistent,
			wantStorageType:       "filesystem",
			wantFilesystemStorage: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
				Fluentbit: &loggingService.Fluentbit{
					ContainerLogging: true,
					StorageProfile:   test.profile,
					Aggregator:       &loggingService.FluentbitAggregator{},
				},
			}}
			configMap, err := forwarderConfigMap(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
			if err != nil {
				t.Fatalf("render Fluent Bit forwarder ConfigMap: %v", err)
			}

			input := configMap.Data["input-containerd.conf"]
			if !strings.Contains(input, "DB                 /fluent-bit/state/containers.db") ||
				!strings.Contains(input, "storage.type       "+test.wantStorageType) {
				t.Errorf("unexpected container input configuration:\n%s", input)
			}
			hasStoragePath := strings.Contains(configMap.Data["fluent-bit.conf"], "storage.path")
			if hasStoragePath != test.wantFilesystemStorage {
				t.Errorf("filesystem storage path present = %v, want %v", hasStoragePath, test.wantFilesystemStorage)
			}
		})
	}
}
