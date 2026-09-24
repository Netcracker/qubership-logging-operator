package fluentbit_forwarder_aggregator

import (
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestHAFluentReconciler() *HAFluentReconciler {
	return &HAFluentReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Log: util.Logger("test-ha-fluent"),
		},
	}
}

func TestAggregatorHTTPOutputTimestampConfiguration(t *testing.T) {
	t.Run("uses the root container timestamp", testAggregatorDefaultHTTPTimestamp)
	for name, extraParams := range map[string]string{
		"custom value": "JSON_DATE_KEY custom_timestamp",
		"disabled":     "json_date_key false",
		"empty":        "json_date_key",
		"duplicated":   "json_date_key first\njson_date_key second",
	} {
		t.Run("rejects "+name+" json_date_key for the default URI", func(t *testing.T) {
			testAggregatorRejectsDefaultJSONDateKey(t, extraParams)
		})
	}
	t.Run("preserves custom URI timestamp configuration", testAggregatorCustomHTTPTimestamp)
	t.Run("preserves disabled json_date_key with a custom URI", testAggregatorDisabledCustomJSONDateKey)
}

func newAggregatorHTTPTestLoggingService(uri, extraParams string) *loggingService.LoggingService {
	return &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				Aggregator: &loggingService.FluentbitAggregator{
					Output: &loggingService.OutputFluentbit{
						Http: &loggingService.HttpFluentbit{
							Enabled:     true,
							Uri:         uri,
							ExtraParams: extraParams,
						},
					},
				},
			},
		},
	}
}

func renderAggregatorHTTPOutput(t *testing.T, uri, extraParams string) string {
	t.Helper()
	configMap, err := aggregatorConfigMap(newAggregatorHTTPTestLoggingService(uri, extraParams), util.DynamicParameters{})
	if err != nil {
		t.Fatalf("failed to render aggregator ConfigMap: %v", err)
	}
	return configMap.Data["output-http.conf"]
}

func assertAggregatorOutputContains(t *testing.T, output, expected, message string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Error(message)
	}
}

func assertAggregatorOutputExcludes(t *testing.T, output, unexpected, message string) {
	t.Helper()
	if strings.Contains(output, unexpected) {
		t.Error(message)
	}
}

func testAggregatorDefaultHTTPTimestamp(t *testing.T) {
	output := renderAggregatorHTTPOutput(t, "", "")
	assertAggregatorOutputContains(t, output, "_time_field=time", "expected the default HTTP URI to use the root time field")
	assertAggregatorOutputExcludes(t, output, "ignore_fields=time", "did not expect VictoriaLogs ingestion to ignore its configured time field")
	assertAggregatorOutputContains(t, output, "_stream_fields=namespace,container", "expected the default HTTP URI to use namespace and container stream fields")
	assertAggregatorOutputContains(t, output, "json_date_key          false", "expected HTTP output not to generate a redundant timestamp field")
}

func testAggregatorRejectsDefaultJSONDateKey(t *testing.T, extraParams string) {
	_, err := aggregatorConfigMap(newAggregatorHTTPTestLoggingService("", extraParams), util.DynamicParameters{})
	if err == nil || !strings.Contains(err.Error(), "must not set json_date_key") {
		t.Fatalf("expected an operator-managed json_date_key error, got: %v", err)
	}
}

func testAggregatorCustomHTTPTimestamp(t *testing.T) {
	const customURI = "/insert/jsonline?_stream_fields=custom&_msg_field=message&_time_field=date"
	output := renderAggregatorHTTPOutput(t, customURI, "json_date_key date")
	assertAggregatorOutputContains(t, output, "uri                    "+customURI, "expected the custom HTTP URI to be preserved")
	assertAggregatorOutputContains(t, output, "json_date_key date", "expected the custom json_date_key to be preserved")
	assertAggregatorOutputExcludes(t, output, "json_date_key          false", "did not expect the operator-managed json_date_key with a custom URI")
	assertAggregatorOutputExcludes(t, output, "ignore_fields=time", "did not expect the operator-managed ignored fields with a custom URI")
}

func testAggregatorDisabledCustomJSONDateKey(t *testing.T) {
	const customURI = "/insert/jsonline?_stream_fields=custom&_msg_field=message"
	output := renderAggregatorHTTPOutput(t, customURI, "json_date_key false")
	assertAggregatorOutputContains(t, output, "json_date_key false", "expected the disabled custom json_date_key to be preserved")
}

func TestHAFluentEqual(t *testing.T) {
	r := newTestHAFluentReconciler()

	t.Run("same data and labels returns true", func(t *testing.T) {
		a := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "fluent"}},
			Data:       map[string]string{"key": "value"},
		}
		b := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "fluent"}},
			Data:       map[string]string{"key": "value"},
		}
		if !r.Equal(a, b) {
			t.Error("expected equal for same data and labels")
		}
	})

	t.Run("different data returns false", func(t *testing.T) {
		a := &corev1.ConfigMap{Data: map[string]string{"key": "value1"}}
		b := &corev1.ConfigMap{Data: map[string]string{"key": "value2"}}
		if r.Equal(a, b) {
			t.Error("expected not equal for different data")
		}
	})

	t.Run("different labels returns false (HA-fluent checks labels)", func(t *testing.T) {
		a := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"env": "prod"}},
			Data:       map[string]string{"key": "value"},
		}
		b := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"env": "dev"}},
			Data:       map[string]string{"key": "value"},
		}
		if r.Equal(a, b) {
			t.Error("HA-fluent Equal should detect label changes, but it didn't")
		}
	})
}

func TestParsedFieldsProtectReservedFields(t *testing.T) {
	configMap, err := aggregatorConfigMap(&loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{Aggregator: &loggingService.FluentbitAggregator{}},
		},
	}, util.DynamicParameters{})
	if err != nil {
		t.Fatalf("failed to render aggregator ConfigMap: %v", err)
	}

	enrichConfig := strings.Join(strings.Fields(configMap.Data["filter-enrich-fields.conf"]), " ")
	for _, rule := range []string{
		"Hard_rename namespace parsed_namespace",
		"Hard_rename pod parsed_pod",
		"Hard_rename container parsed_container",
		"Hard_rename source parsed_source",
		"Hard_rename labels parsed_labels",
		"Hard_rename log parsed_log",
		"Hard_rename time parsed_time",
		"Hard_rename level parsed_level",
		"Hard_rename parse_status parsed_parse_status",
		"Hard_rename source_level parsed_source_level",
	} {
		if !strings.Contains(enrichConfig, rule) {
			t.Errorf("missing reserved field rule %q", rule)
		}
	}
	if strings.Contains(enrichConfig, "Add_prefix parsed_") {
		t.Error("application fields without protected names must keep their original names")
	}

	hideIndex := strings.Index(enrichConfig, "Operation nest Wildcard namespace")
	applicationIndex := strings.Index(enrichConfig, "Nested_under log_parsed")
	renameIndex := strings.Index(enrichConfig, "Hard_rename namespace parsed_namespace")
	restoreIndex := strings.LastIndex(enrichConfig, "Nested_under _record_metadata")
	if hideIndex < 0 || hideIndex >= applicationIndex || applicationIndex >= renameIndex || renameIndex >= restoreIndex {
		t.Error("protected fields must be hidden, application fields extracted and renamed, then protected fields restored")
	}

	levelConfig := strings.Join(strings.Fields(configMap.Data["filter-nonsupported-levels.conf"]), " ")
	if !strings.Contains(levelConfig, "Rename parsed_source_level source_level") {
		t.Error("source_level must be restored without overwriting the normalized value")
	}
}

func TestParserSuccessUsesPreserveKeyOff(t *testing.T) {
	configMap, err := aggregatorConfigMap(&loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{Aggregator: &loggingService.FluentbitAggregator{}},
		},
	}, util.DynamicParameters{})
	if err != nil {
		t.Fatalf("failed to render aggregator ConfigMap: %v", err)
	}

	genericConfig := strings.Join(strings.Fields(configMap.Data["filter-generic.conf"]), " ")
	for _, expected := range []string{
		"Copy log _parser_input",
		"Key_Name _parser_input",
		"Preserve_Key Off",
	} {
		if !strings.Contains(genericConfig, expected) {
			t.Errorf("generic parser pipeline is missing %q", expected)
		}
	}
	if strings.Contains(genericConfig, "Preserve_Key On") {
		t.Error("generic parsers must remove original_log after successful parsing")
	}

	statusConfig := configMap.Data["filter-validate.conf"] + configMap.Data["filter-post-generic.conf"]
	if strings.Count(statusConfig, "Key_does_not_exist _parser_input") != 2 {
		t.Error("klog and generic parser success must be detected from the removed _parser_input field")
	}
	if !strings.Contains(configMap.Data["filter-enrich-fields.conf"], "Preserve_Key    Off") {
		t.Error("klog parsers must remove original_log after successful parsing")
	}
	for name, content := range configMap.Data {
		if strings.Contains(content, "count_fields") {
			t.Errorf("%s still uses field-count parsing detection", name)
		}
	}
}
