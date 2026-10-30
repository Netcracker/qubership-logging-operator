package fluentbit_forwarder_aggregator

import (
	"regexp"
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
	normalizedGenericConfig := strings.Join(strings.Fields(genericConfig), " ")
	if !strings.Contains(normalizedGenericConfig,
		`Match_Regex ^pods(?!.*opensearch-\d{1,2}_).* Key_Name _parser_input Parser qubership`) {
		t.Errorf("expected the generalized Qubership parser to exclude OpenSearch tags, got:\n%s", genericConfig)
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
	for _, configName := range []string{"filter-validate.conf", "filter-post-generic.conf"} {
		config := string(secret.Data[configName])
		normalizedConfig := strings.Join(strings.Fields(config), " ")
		cleanupIndex := strings.Index(normalizedConfig, "Condition Key_exists __qubership_candidate")
		parseIndex := strings.Index(normalizedConfig, "kv_parse")
		if cleanupIndex == -1 || parseIndex == -1 || cleanupIndex > parseIndex ||
			!strings.Contains(normalizedConfig, "Condition Key_value_does_not_match log") ||
			!strings.Contains(normalizedConfig, `^\[\d{4}-\d{2}-\d{2}[Tt\x20]\d{2}:\d{2}:\d{2}`) ||
			!strings.Contains(normalizedConfig, "Remove __qubership_candidate") {
			t.Errorf("expected non-Qubership marker cleanup before key-value parsing in %s, got:\n%s", configName, config)
		}
	}
	for _, expected := range []string{
		"Match pods* Condition Key_exists __qubership_candidate Set parse_format qubership",
		"Match pods* Condition Key_value_equals parse_format qubership",
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
	if strings.Contains(normalizedPostGenericConfig,
		"Match_regex (pods|klog).* Condition Key_exists __qubership_candidate Set parse_format qubership") {
		t.Errorf("unexpected Qubership classification for rewritten klog records:\n%s", postGenericConfig)
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
	headerGuardIndex := strings.Index(script, "if header_end == nil then")
	pairIndex := strings.Index(script, "string.match(s, regex_kv, header_end + 1)")
	if headerGuardIndex == -1 || pairIndex == -1 || headerGuardIndex > pairIndex {
		t.Errorf("expected key-value fields to be read only after the Qubership header, got:\n%s", script)
	}
}

func TestAggregatorLevelFilterMatchesRoutingState(t *testing.T) {
	const routingMatch = "Match_regex       ^(?!out_(audit|k8s_event|nginx|access|int|pods|system|default)$).*"

	tests := []struct {
		name   string
		output *loggingService.OutputFluentbit
		want   string
		absent string
	}{
		{
			name:   "no HTTP output",
			want:   "Match             *",
			absent: "Match_regex",
		},
		{
			name: "HTTP output with routing disabled",
			output: &loggingService.OutputFluentbit{Http: &loggingService.HttpFluentbit{
				Enabled: true, Host: "vmauth", Port: 8427, Uri: "/insert/jsonline",
				Routing: &loggingService.FluentbitHTTPRouting{Enabled: false},
			}},
			want:   "Match             *",
			absent: "Match_regex",
		},
		{
			name: "HTTP output with routing enabled",
			output: &loggingService.OutputFluentbit{Http: &loggingService.HttpFluentbit{
				Enabled: true, Host: "vmauth", Port: 8427, Uri: "/insert/jsonline",
				Routing: &loggingService.FluentbitHTTPRouting{Enabled: true},
			}},
			want:   routingMatch,
			absent: "Match             *",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
				Fluentbit: &loggingService.Fluentbit{
					ContainerLogging: true,
					Aggregator:       &loggingService.FluentbitAggregator{Output: test.output},
				},
			}}
			configSecret, err := aggregatorConfigSecret(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"},
				aggregatorOutputCredentials{})
			if err != nil {
				t.Fatalf("render Fluent Bit aggregator config Secret: %v", err)
			}
			filter := string(configSecret.Data["filter-nonsupported-levels.conf"])
			if !strings.Contains(filter, test.want) || strings.Contains(filter, test.absent) {
				t.Errorf("expected %q without %q in the level filter, got:\n%s", test.want, test.absent, filter)
			}
		})
	}
}

func TestForwarderConfigMapUsesLanguageMultilineParsers(t *testing.T) {
	cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
		Fluentbit: &loggingService.Fluentbit{
			ContainerLogging: true,
			Aggregator:       &loggingService.FluentbitAggregator{},
		},
	}}
	configMap, err := forwarderConfigMap(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
	if err != nil {
		t.Fatalf("render Fluent Bit forwarder ConfigMap: %v", err)
	}

	if !strings.Contains(configMap.Data["filter-concat.conf"],
		"multiline.parser       multiline_qubership, java, go, multiline_python") {
		t.Errorf("expected the application multiline parser chain, got:\n%s",
			configMap.Data["filter-concat.conf"])
	}
}

func TestForwarderConfigMapMemoryLimits(t *testing.T) {
	tests := []struct {
		name             string
		inputMemBufLimit string
		wantMemBufLimit  string
	}{
		{name: "default", wantMemBufLimit: "10M"},
		{name: "custom", inputMemBufLimit: "25M", wantMemBufLimit: "25M"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
				Fluentbit: &loggingService.Fluentbit{
					ContainerLogging:   true,
					SystemAuditLogging: true,
					InputMemBufLimit:   test.inputMemBufLimit,
					StorageProfile:     loggingService.FluentbitStorageProfileNodePersistent,
					Aggregator:         &loggingService.FluentbitAggregator{},
				},
			}}
			configMap, err := forwarderConfigMap(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
			if err != nil {
				t.Fatalf("render Fluent Bit forwarder ConfigMap: %v", err)
			}

			for _, name := range []string{"input-containerd.conf", "input-audit.conf"} {
				if !strings.Contains(configMap.Data[name], "Mem_Buf_Limit      "+test.wantMemBufLimit) {
					t.Errorf("expected the memory buffer limit %s in %s, got:\n%s",
						test.wantMemBufLimit, name, configMap.Data[name])
				}
			}
			if got := strings.Count(configMap.Data["filter-concat.conf"], "emitter_mem_buf_limit  32MB"); got != 2 {
				t.Errorf("expected 2 emitters limited to 32MB, got %d:\n%s", got, configMap.Data["filter-concat.conf"])
			}
			if !strings.Contains(configMap.Data["fluent-bit.conf"], "storage.max_chunks_up                64") {
				t.Errorf("expected 64 chunks up for the filesystem buffer, got:\n%s", configMap.Data["fluent-bit.conf"])
			}
		})
	}
}

func TestAggregatorConfigMemoryLimits(t *testing.T) {
	tests := []struct {
		name             string
		inputMemBufLimit string
		wantMemBufLimit  string
	}{
		{name: "default", wantMemBufLimit: "200M"},
		{name: "custom", inputMemBufLimit: "300M", wantMemBufLimit: "300M"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := renderAggregatorConfigData(t, &loggingService.FluentbitAggregator{
				InputMemBufLimit: test.inputMemBufLimit,
			})

			input := data["input-forward.conf"]
			for _, expected := range []string{
				"mem_buf_limit     " + test.wantMemBufLimit,
				"Buffer_Chunk_Size 1M",
			} {
				if !strings.Contains(input, expected) {
					t.Errorf("expected %q in the forward input, got:\n%s", expected, input)
				}
			}
			if !strings.Contains(data["fluent-bit.conf"], "storage.max_chunks_up                256") {
				t.Errorf("expected 256 chunks up, got:\n%s", data["fluent-bit.conf"])
			}
			for _, unexpected := range []string{"label_field          src_ip", "label_field          dst_ip"} {
				if strings.Contains(data["filter-unparsed-log-counter.conf"], unexpected) {
					t.Errorf("unexpected %q in the calico metric filter, got:\n%s", unexpected,
						data["filter-unparsed-log-counter.conf"])
				}
			}
		})
	}
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

func TestAggregatorRoutingFollowsLevelNormalizationAndFormatDetection(t *testing.T) {
	data := renderAggregatorConfigData(t, &loggingService.FluentbitAggregator{
		Output: &loggingService.OutputFluentbit{
			Http: &loggingService.HttpFluentbit{
				Enabled: true,
				Routing: &loggingService.FluentbitHTTPRouting{Enabled: true},
			},
		},
	})
	mainConfig := data["fluent-bit.conf"]
	formatDetection := strings.Index(mainConfig, "filter-define-format.conf")
	routing := strings.Index(mainConfig, "filter-http-routing.conf")
	normalization := strings.Index(mainConfig, "filter-nonsupported-levels.conf")
	// Routed records re-enter the filter chain with out_* tags. The level filter skips those tags, so
	// routing must run after normalization, and after format detection, which matches the input tags.
	if formatDetection < 0 || routing < 0 || normalization < 0 ||
		normalization >= formatDetection || formatDetection >= routing {
		t.Errorf("level normalization, format detection, and HTTP routing are in the wrong order:\n%s", mainConfig)
	}
	if !strings.Contains(data["filter-http-routing.conf"], "Name                   rewrite_tag") {
		t.Error("HTTP routing configuration has no rewrite_tag filter")
	}
	if strings.Contains(data["output-http.conf"], "rewrite_tag") {
		t.Error("HTTP output configuration still contains routing filters")
	}

	script := data["update_level_syslog.lua"]
	if strings.Contains(script, `record["detected_level"] ~= nil`) {
		t.Error("detected_level is used as a processing marker")
	}
	if strings.Contains(script, "routing_tags") {
		t.Error("level normalization depends on routing tags")
	}
}
