package fluentbit_forwarder_aggregator

import (
	"bytes"
	"os"
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAggregatorCollisionLuaMatchesStandalone(t *testing.T) {
	aggregatorLua, err := os.ReadFile("aggregator.configmap/conf.d/lua_scripts/count_fields.lua")
	if err != nil {
		t.Fatalf("failed to read aggregator collision Lua: %v", err)
	}
	standaloneLua, err := os.ReadFile("../fluentbit/fluentbit.configmap/conf.d/lua_scripts/count_fields.lua")
	if err != nil {
		t.Fatalf("failed to read standalone collision Lua: %v", err)
	}
	if !bytes.Equal(aggregatorLua, standaloneLua) {
		t.Fatal("aggregator and standalone collision handling must stay identical")
	}
}

func newTestHAFluentReconciler() *HAFluentReconciler {
	return &HAFluentReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Log: util.Logger("test-ha-fluent"),
		},
	}
}

func TestAggregatorHTTPOutputTimestampConfiguration(t *testing.T) {
	newLoggingService := func(uri, extraParams string) *loggingService.LoggingService {
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

	t.Run("uses the root container timestamp", func(t *testing.T) {
		configMap, err := aggregatorConfigMap(newLoggingService("", ""), util.DynamicParameters{})
		if err != nil {
			t.Fatalf("failed to render aggregator ConfigMap: %v", err)
		}

		output := configMap.Data["output-http.conf"]
		if !strings.Contains(output, "_time_field=time") {
			t.Error("expected the default HTTP URI to use the root time field")
		}
		if strings.Contains(output, "ignore_fields=time") {
			t.Error("did not expect VictoriaLogs ingestion to ignore its configured time field")
		}
		if !strings.Contains(output, "_stream_fields=namespace,container") {
			t.Error("expected the default HTTP URI to use namespace and container stream fields")
		}
		if !strings.Contains(output, "json_date_key          false") {
			t.Error("expected HTTP output not to generate a redundant timestamp field")
		}
	})

	for name, extraParams := range map[string]string{
		"custom value": "JSON_DATE_KEY custom_timestamp",
		"disabled":     "json_date_key false",
		"empty":        "json_date_key",
		"duplicated":   "json_date_key first\njson_date_key second",
	} {
		t.Run("rejects "+name+" json_date_key for the default URI", func(t *testing.T) {
			_, err := aggregatorConfigMap(newLoggingService("", extraParams), util.DynamicParameters{})
			if err == nil || !strings.Contains(err.Error(), "must not set json_date_key") {
				t.Fatalf("expected an operator-managed json_date_key error, got: %v", err)
			}
		})
	}

	t.Run("preserves custom URI timestamp configuration", func(t *testing.T) {
		const customURI = "/insert/jsonline?_stream_fields=custom&_msg_field=message&_time_field=date"
		configMap, err := aggregatorConfigMap(newLoggingService(customURI, "json_date_key date"), util.DynamicParameters{})
		if err != nil {
			t.Fatalf("failed to render custom aggregator HTTP output: %v", err)
		}

		output := configMap.Data["output-http.conf"]
		if !strings.Contains(output, "uri                    "+customURI) {
			t.Error("expected the custom HTTP URI to be preserved")
		}
		if !strings.Contains(output, "json_date_key date") {
			t.Error("expected the custom json_date_key to be preserved")
		}
		if strings.Contains(output, "json_date_key          false") {
			t.Error("did not expect the operator-managed json_date_key with a custom URI")
		}
		if strings.Contains(output, "ignore_fields=time") {
			t.Error("did not expect the operator-managed ignored fields with a custom URI")
		}
	})

	t.Run("preserves disabled json_date_key with a custom URI", func(t *testing.T) {
		const customURI = "/insert/jsonline?_stream_fields=custom&_msg_field=message"
		configMap, err := aggregatorConfigMap(newLoggingService(customURI, "json_date_key false"), util.DynamicParameters{})
		if err != nil {
			t.Fatalf("failed to render disabled custom json_date_key: %v", err)
		}
		if !strings.Contains(configMap.Data["output-http.conf"], "json_date_key false") {
			t.Error("expected the disabled custom json_date_key to be preserved")
		}
	})
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
