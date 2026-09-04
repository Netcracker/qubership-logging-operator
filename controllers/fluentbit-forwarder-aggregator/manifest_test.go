package fluentbit_forwarder_aggregator

import (
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
)

func assertRawLogFallbackAfterPlaceholder(t *testing.T, normalizedConfig, originalConfig string) {
	t.Helper()
	placeholderIndex := strings.Index(normalizedConfig, `Set short_message "<Empty message>"`)
	copyIndex := strings.Index(normalizedConfig, "Copy log short_message")
	if placeholderIndex == -1 || copyIndex <= placeholderIndex {
		t.Errorf("expected the raw-log fallback after the Qubership placeholder, got:\n%s", originalConfig)
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

func TestAggregatorGeneralizedQubershipParserAndEmptyMessagePlaceholder(t *testing.T) {
	cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
		Fluentbit: &loggingService.Fluentbit{Aggregator: &loggingService.FluentbitAggregator{}},
	}}
	secret, err := aggregatorConfigSecret(cr, util.DynamicParameters{}, aggregatorOutputCredentials{})
	if err != nil {
		t.Fatalf("render Fluent Bit aggregator config Secret: %v", err)
	}

	genericConfig := string(secret.Data["filter-generic.conf"])
	if !strings.Contains(genericConfig, "Parser         qubership") {
		t.Errorf("expected the generalized Qubership parser in the generic filter, got:\n%s", genericConfig)
	}
	if !strings.Contains(genericConfig, "Parser         json") {
		t.Errorf("expected the generic JSON parser to remain enabled, got:\n%s", genericConfig)
	}
	if strings.Contains(genericConfig, "Parser         java") {
		t.Errorf("unexpected Java parser in the generic filter:\n%s", genericConfig)
	}
	parsersConfig := string(secret.Data["parsers.conf"])
	if strings.Contains(parsersConfig, "\n    Name        java\n") {
		t.Errorf("unexpected separate Java parser definition:\n%s", parsersConfig)
	}
	if !strings.Contains(parsersConfig, `(?<__qubership_candidate>\[)`) {
		t.Errorf("expected the Qubership parser to emit a match marker, got:\n%s", parsersConfig)
	}
	if !strings.Contains(parsersConfig, `(?<short_message>[\s\S]*)`) {
		t.Errorf("expected the Qubership parser to allow an empty message, got:\n%s", parsersConfig)
	}
	if !strings.Contains(parsersConfig, `\]\s*)*(?<short_message>`) {
		t.Errorf("expected the generalized Qubership parser to allow zero key-value pairs, got:\n%s", parsersConfig)
	}

	postGenericConfig := string(secret.Data["filter-post-generic.conf"])
	normalizedPostGenericConfig := strings.Join(strings.Fields(postGenericConfig), " ")
	for _, expected := range []string{
		"Condition Key_exists __qubership_candidate",
		"Condition Key_value_equals parse_format qubership",
		"Condition Key_does_not_exist short_message",
		`Set short_message "<Empty message>"`,
		"Copy log short_message",
		"Remove_key __qubership_candidate",
	} {
		if !strings.Contains(normalizedPostGenericConfig, expected) {
			t.Errorf("expected %q in the post-generic config, got:\n%s", expected, postGenericConfig)
		}
	}
	if strings.Contains(postGenericConfig, "qubership_short_message_missing") {
		t.Errorf("unexpected missing-message marker in the post-generic config:\n%s", postGenericConfig)
	}
	for _, unexpected := range []string{
		"Set parse_format java",
		"Condition Key_exists tenant_id",
		"Condition Key_exists thread",
		"Condition Key_exists request_id",
		"Condition Key_exists class",
	} {
		if strings.Contains(normalizedPostGenericConfig, unexpected) {
			t.Errorf("unexpected %q in the post-generic config:\n%s", unexpected, postGenericConfig)
		}
	}
	assertRawLogFallbackAfterPlaceholder(t, normalizedPostGenericConfig, postGenericConfig)
}

func TestAggregatorQubershipKeyValueParserGuard(t *testing.T) {
	cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
		Fluentbit: &loggingService.Fluentbit{Aggregator: &loggingService.FluentbitAggregator{}},
	}}
	secret, err := aggregatorConfigSecret(cr, util.DynamicParameters{}, aggregatorOutputCredentials{})
	if err != nil {
		t.Fatalf("render Fluent Bit aggregator config Secret: %v", err)
	}
	script := string(secret.Data["parse_key_value.lua"])

	if !strings.Contains(script, `if record["__qubership_candidate"] == nil then`) {
		t.Errorf("expected key-value parsing to require a Qubership parser marker, got:\n%s", script)
	}
}
