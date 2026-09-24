package fluentbit

import (
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestFluentbitReconciler() *FluentbitReconciler {
	return &FluentbitReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Log: util.Logger("test-fluentbit"),
		},
	}
}

func TestFluentbitHTTPOutputTimestampConfiguration(t *testing.T) {
	t.Run("uses the root container timestamp", testFluentbitDefaultHTTPTimestamp)
	for name, extraParams := range map[string]string{
		"custom value": "JSON_DATE_KEY custom_timestamp",
		"disabled":     "json_date_key false",
		"empty":        "json_date_key",
		"duplicated":   "json_date_key first\njson_date_key second",
	} {
		t.Run("rejects "+name+" json_date_key for the default URI", func(t *testing.T) {
			testFluentbitRejectsDefaultJSONDateKey(t, extraParams)
		})
	}
	t.Run("preserves custom URI timestamp configuration", testFluentbitCustomHTTPTimestamp)
	t.Run("preserves disabled json_date_key with a custom URI", testFluentbitDisabledCustomJSONDateKey)
}

func newFluentbitHTTPTestLoggingService(uri, extraParams string) *loggingService.LoggingService {
	return &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				Output: &loggingService.OutputFluentbit{
					Http: &loggingService.HttpFluentbit{
						Enabled:     true,
						Uri:         uri,
						ExtraParams: extraParams,
					},
				},
			},
		},
	}
}

func renderFluentbitHTTPOutput(t *testing.T, uri, extraParams string) string {
	t.Helper()
	configMap, err := fluentbitConfigMap(newFluentbitHTTPTestLoggingService(uri, extraParams), util.DynamicParameters{})
	if err != nil {
		t.Fatalf("failed to render Fluent Bit ConfigMap: %v", err)
	}
	return configMap.Data["output-http.conf"]
}

func assertFluentbitOutputContains(t *testing.T, output, expected, message string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Error(message)
	}
}

func assertFluentbitOutputExcludes(t *testing.T, output, unexpected, message string) {
	t.Helper()
	if strings.Contains(output, unexpected) {
		t.Error(message)
	}
}

func testFluentbitDefaultHTTPTimestamp(t *testing.T) {
	output := renderFluentbitHTTPOutput(t, "", "")
	assertFluentbitOutputContains(t, output, "_time_field=time", "expected the default HTTP URI to use the root time field")
	assertFluentbitOutputExcludes(t, output, "ignore_fields=time", "did not expect VictoriaLogs ingestion to ignore its configured time field")
	assertFluentbitOutputContains(t, output, "_stream_fields=namespace,container", "expected the default HTTP URI to use namespace and container stream fields")
	assertFluentbitOutputContains(t, output, "json_date_key          false", "expected HTTP output not to generate a redundant timestamp field")
}

func testFluentbitRejectsDefaultJSONDateKey(t *testing.T, extraParams string) {
	_, err := fluentbitConfigMap(newFluentbitHTTPTestLoggingService("", extraParams), util.DynamicParameters{})
	if err == nil || !strings.Contains(err.Error(), "must not set json_date_key") {
		t.Fatalf("expected an operator-managed json_date_key error, got: %v", err)
	}
}

func testFluentbitCustomHTTPTimestamp(t *testing.T) {
	const customURI = "/insert/jsonline?_stream_fields=custom&_msg_field=message&_time_field=date"
	output := renderFluentbitHTTPOutput(t, customURI, "json_date_key date")
	assertFluentbitOutputContains(t, output, "uri                    "+customURI, "expected the custom HTTP URI to be preserved")
	assertFluentbitOutputContains(t, output, "json_date_key date", "expected the custom json_date_key to be preserved")
	assertFluentbitOutputExcludes(t, output, "json_date_key          false", "did not expect the operator-managed json_date_key with a custom URI")
	assertFluentbitOutputExcludes(t, output, "ignore_fields=time", "did not expect the operator-managed ignored fields with a custom URI")
}

func testFluentbitDisabledCustomJSONDateKey(t *testing.T) {
	const customURI = "/insert/jsonline?_stream_fields=custom&_msg_field=message"
	output := renderFluentbitHTTPOutput(t, customURI, "json_date_key false")
	assertFluentbitOutputContains(t, output, "json_date_key false", "expected the disabled custom json_date_key to be preserved")
}

func TestFluentbitEqual(t *testing.T) {
	r := newTestFluentbitReconciler()

	t.Run("same data returns true", func(t *testing.T) {
		a := &corev1.ConfigMap{Data: map[string]string{"key": "value"}}
		b := &corev1.ConfigMap{Data: map[string]string{"key": "value"}}
		if !r.Equal(a, b) {
			t.Error("expected equal for same data")
		}
	})

	t.Run("different data returns false", func(t *testing.T) {
		a := &corev1.ConfigMap{Data: map[string]string{"key": "value1"}}
		b := &corev1.ConfigMap{Data: map[string]string{"key": "value2"}}
		if r.Equal(a, b) {
			t.Error("expected not equal for different data")
		}
	})

	t.Run("different binary data returns false", func(t *testing.T) {
		a := &corev1.ConfigMap{BinaryData: map[string][]byte{"key": {1, 2}}}
		b := &corev1.ConfigMap{BinaryData: map[string][]byte{"key": {3, 4}}}
		if r.Equal(a, b) {
			t.Error("expected not equal for different binary data")
		}
	})

	t.Run("different labels still returns true (fluentbit ignores labels)", func(t *testing.T) {
		a := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"env": "prod"}},
			Data:       map[string]string{"key": "value"},
		}
		b := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"env": "dev"}},
			Data:       map[string]string{"key": "value"},
		}
		if !r.Equal(a, b) {
			t.Error("fluentbit Equal should ignore labels, but it didn't")
		}
	})
}

func TestParsedFieldsProtectReservedFields(t *testing.T) {
	configMap, err := fluentbitConfigMap(&loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{Fluentbit: &loggingService.Fluentbit{}},
	}, util.DynamicParameters{})
	if err != nil {
		t.Fatalf("failed to render Fluent Bit ConfigMap: %v", err)
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
