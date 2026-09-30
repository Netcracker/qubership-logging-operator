package fluentbit_forwarder_aggregator

import (
	"regexp"
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
)

func aggregatorMatcherFromConfig(t *testing.T, config string) *regexp.Regexp {
	t.Helper()
	for _, line := range strings.Split(config, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.EqualFold(fields[0], "Match_Regex") {
			matcher, err := regexp.Compile(fields[1])
			if err != nil {
				t.Fatalf("compile output matcher %q: %v", fields[1], err)
			}
			return matcher
		}
	}
	t.Fatal("output configuration has no Match_Regex")
	return nil
}

func renderAggregatorConfigData(t *testing.T, aggregator *loggingService.FluentbitAggregator) map[string]string {
	t.Helper()
	cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
		Fluentbit: &loggingService.Fluentbit{
			GraylogOutput: aggregator.GraylogOutput,
			Aggregator:    aggregator,
		},
	}}
	secret, err := aggregatorConfigSecret(cr, util.DynamicParameters{}, aggregatorOutputCredentials{})
	if err != nil {
		t.Fatalf("render aggregator config Secret: %v", err)
	}
	data := make(map[string]string, len(secret.Data))
	for key, value := range secret.Data {
		data[key] = string(value)
	}
	return data
}

func TestAggregatorRoutedOutputMatchers(t *testing.T) {
	output := &loggingService.OutputFluentbit{
		Loki: &loggingService.LokiFluentbit{Enabled: true},
		Http: &loggingService.HttpFluentbit{
			Enabled: true,
			Routing: &loggingService.FluentbitHTTPRouting{Enabled: true},
		},
		Otel: &loggingService.OtelFluentbit{Enabled: true},
	}
	data := renderAggregatorConfigData(t, &loggingService.FluentbitAggregator{
		GraylogOutput: true,
		Output:        output,
	})

	for _, file := range []string{"output-graylog.conf", "output-loki.conf", "output-opentelemetry.conf"} {
		t.Run(file, func(t *testing.T) {
			matcher := aggregatorMatcherFromConfig(t, data[file])
			for _, tag := range []string{
				"out_audit", "out_system", "out_pods", "out_nginx", "out_k8s_event", "out_access", "out_int",
				"audit.var.log", "system.var.log", "pods.var.log", "klog.var.log",
			} {
				if !matcher.MatchString(tag) {
					t.Errorf("matcher does not select %q", tag)
				}
			}
			for _, tag := range []string{"out_default", "out_custom", "out_audit_custom"} {
				if matcher.MatchString(tag) {
					t.Errorf("matcher unexpectedly selects custom tag %q", tag)
				}
			}
		})
	}
}

func TestAggregatorLokiMatcherSelectsKlogWithoutRouting(t *testing.T) {
	data := renderAggregatorConfigData(t, &loggingService.FluentbitAggregator{
		Output: &loggingService.OutputFluentbit{
			Loki: &loggingService.LokiFluentbit{Enabled: true},
		},
	})

	if !aggregatorMatcherFromConfig(t, data["output-loki.conf"]).MatchString("klog.var.log") {
		t.Error("Loki matcher does not select klog without routing")
	}
}

func TestForwarderConfigMapStorageProfiles(t *testing.T) {
	tests := []struct {
		name                  string
		profile               string
		wantStorageType       string
		wantEmitterStorage    string
		wantFilesystemStorage bool
	}{
		{
			name:               "memory only",
			profile:            loggingService.FluentbitStorageProfileMemoryOnly,
			wantStorageType:    "memory",
			wantEmitterStorage: "memory",
		},
		{
			name:               "persistent offsets",
			profile:            loggingService.FluentbitStorageProfilePersistentOffsets,
			wantStorageType:    "memory",
			wantEmitterStorage: "memory",
		},
		{
			name:                  "node persistent",
			profile:               loggingService.FluentbitStorageProfileNodePersistent,
			wantStorageType:       "filesystem",
			wantEmitterStorage:    "filesystem",
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
			wantEmitterStorage := "emitter_storage.type   " + test.wantEmitterStorage
			if strings.Count(configMap.Data["filter-concat.conf"], wantEmitterStorage) != 2 {
				t.Errorf("expected %q for both multiline emitters, got:\n%s", wantEmitterStorage,
					configMap.Data["filter-concat.conf"])
			}
		})
	}
}
