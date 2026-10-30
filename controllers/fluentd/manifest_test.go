package fluentd

import (
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func renderFluentdConfigData(t *testing.T, fluentd *loggingService.Fluentd) map[string]string {
	t.Helper()
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging"},
		Spec:       loggingService.LoggingServiceSpec{Fluentd: fluentd},
	}
	secret, err := fluentdConfigSecret(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"}, outputCredentials{})
	if err != nil {
		t.Fatalf("render FluentD config Secret: %v", err)
	}
	data := make(map[string]string, len(secret.Data))
	for key, value := range secret.Data {
		data[key] = string(value)
	}
	return data
}

func TestFluentdGraylogBufferLimit(t *testing.T) {
	tests := []struct {
		name           string
		fileStorage    bool
		totalLimitSize string
		want           string
	}{
		{name: "memory buffer default", want: "128MB"},
		{name: "file buffer default", fileStorage: true, want: "512MB"},
		{name: "memory buffer from the CR", totalLimitSize: "64MB", want: "64MB"},
		{name: "file buffer from the CR", fileStorage: true, totalLimitSize: "2GB", want: "2GB"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := renderFluentdConfigData(t, &loggingService.Fluentd{
				GraylogOutput:  true,
				GraylogHost:    "graylog",
				GraylogPort:    12201,
				FileStorage:    test.fileStorage,
				TotalLimitSize: test.totalLimitSize,
				Output: &loggingService.OutputFluentd{
					Loki: &loggingService.LokiFluentd{},
					Http: &loggingService.HttpFluentd{},
				},
			})
			conf := data["output-graylog.conf"]
			if !strings.Contains(conf, "total_limit_size "+test.want+"\n") {
				t.Errorf("expected the buffer limit %s, got:\n%s", test.want, conf)
			}
		})
	}
}

func TestFluentdLokiBufferLimit(t *testing.T) {
	tests := []struct {
		name        string
		extraParams string
		wantBuffer  bool
	}{
		{name: "default buffer", wantBuffer: true},
		{name: "buffer from extra parameters", extraParams: "<buffer>\n  total_limit_size 32MB\n</buffer>"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := renderFluentdConfigData(t, &loggingService.Fluentd{
				Output: &loggingService.OutputFluentd{Loki: &loggingService.LokiFluentd{
					Enabled:     true,
					Host:        "http://loki:3100",
					ExtraParams: test.extraParams,
				}, Http: &loggingService.HttpFluentd{}},
			})
			conf := data["output-loki.conf"]
			if got := strings.Count(conf, "<buffer"); got != 1 {
				t.Errorf("expected exactly one buffer section, got %d:\n%s", got, conf)
			}
			if hasDefault := strings.Contains(conf, "total_limit_size 128MB"); hasDefault != test.wantBuffer {
				t.Errorf("default buffer present = %v, want %v:\n%s", hasDefault, test.wantBuffer, conf)
			}
		})
	}
}

func TestFluentdHttpBufferLimit(t *testing.T) {
	data := renderFluentdConfigData(t, &loggingService.Fluentd{
		Output: &loggingService.OutputFluentd{Http: &loggingService.HttpFluentd{
			Enabled: true,
			Host:    "http://victorialogs:9428",
		}, Loki: &loggingService.LokiFluentd{}},
	})
	conf := data["output-http.conf"]
	if !strings.Contains(conf, "total_limit_size 128MB") {
		t.Errorf("expected the buffer limit 128MB, got:\n%s", conf)
	}
}

func TestFluentdCalicoMetricHasNoIPAddressLabels(t *testing.T) {
	data := renderFluentdConfigData(t, &loggingService.Fluentd{
		Output: &loggingService.OutputFluentd{Loki: &loggingService.LokiFluentd{}, Http: &loggingService.HttpFluentd{}},
	})
	conf := data["filter-calico-packets.conf"]
	for _, unexpected := range []string{"src_ip ${src_ip}", "dst_ip ${dst_ip}"} {
		if strings.Contains(conf, unexpected) {
			t.Errorf("unexpected label %q in the calico metric, got:\n%s", unexpected, conf)
		}
	}
}
