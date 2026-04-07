package v1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGraylogIsInstall(t *testing.T) {
	t.Run("non-nil returns true", func(t *testing.T) {
		g := &Graylog{}
		if !g.IsInstall() {
			t.Error("expected true for non-nil Graylog")
		}
	})

	t.Run("nil returns false", func(t *testing.T) {
		var g *Graylog
		if g.IsInstall() {
			t.Error("expected false for nil Graylog")
		}
	})
}

func TestFluentdIsInstall(t *testing.T) {
	t.Run("non-nil returns true", func(t *testing.T) {
		f := &Fluentd{}
		if !f.IsInstall() {
			t.Error("expected true for non-nil Fluentd")
		}
	})

	t.Run("nil returns false", func(t *testing.T) {
		var f *Fluentd
		if f.IsInstall() {
			t.Error("expected false for nil Fluentd")
		}
	})
}

func TestFluentbitIsInstall(t *testing.T) {
	t.Run("non-nil returns true", func(t *testing.T) {
		f := &Fluentbit{}
		if !f.IsInstall() {
			t.Error("expected true for non-nil Fluentbit")
		}
	})

	t.Run("nil returns false", func(t *testing.T) {
		var f *Fluentbit
		if f.IsInstall() {
			t.Error("expected false for nil Fluentbit")
		}
	})
}

func TestCloudEventsReaderIsInstall(t *testing.T) {
	t.Run("non-nil returns true", func(t *testing.T) {
		c := &CloudEventsReader{}
		if !c.IsInstall() {
			t.Error("expected true for non-nil CloudEventsReader")
		}
	})

	t.Run("nil returns false", func(t *testing.T) {
		var c *CloudEventsReader
		if c.IsInstall() {
			t.Error("expected false for nil CloudEventsReader")
		}
	})
}

func TestMonitoringAgentLoggingPluginIsInstall(t *testing.T) {
	t.Run("non-nil with InfluxDBMode true", func(t *testing.T) {
		m := &MonitoringAgentLoggingPlugin{InfluxDBMode: true}
		if !m.IsInstall() {
			t.Error("expected true when InfluxDBMode=true")
		}
	})

	t.Run("non-nil with InfluxDBMode false", func(t *testing.T) {
		m := &MonitoringAgentLoggingPlugin{InfluxDBMode: false}
		if m.IsInstall() {
			t.Error("expected false when InfluxDBMode=false")
		}
	})

	t.Run("nil returns false", func(t *testing.T) {
		var m *MonitoringAgentLoggingPlugin
		if m.IsInstall() {
			t.Error("expected false for nil MonitoringAgentLoggingPlugin")
		}
	})
}

func TestGraylogIsForceUpdate(t *testing.T) {
	tests := []struct {
		name     string
		policy   string
		expected bool
	}{
		{"force-update", "force-update", true},
		{"only-create", "only-create", false},
		{"empty string", "", false},
		{"other value", "something", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Graylog{ContentDeployPolicy: tt.policy}
			if g.IsForceUpdate() != tt.expected {
				t.Errorf("IsForceUpdate() with policy %q = %v, want %v", tt.policy, g.IsForceUpdate(), tt.expected)
			}
		})
	}
}

func TestGraylogIsOnlyCreate(t *testing.T) {
	tests := []struct {
		name     string
		policy   string
		expected bool
	}{
		{"only-create", "only-create", true},
		{"force-update", "force-update", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Graylog{ContentDeployPolicy: tt.policy}
			if g.IsOnlyCreate() != tt.expected {
				t.Errorf("IsOnlyCreate() with policy %q = %v, want %v", tt.policy, g.IsOnlyCreate(), tt.expected)
			}
		})
	}
}

func TestToParams(t *testing.T) {
	ls := &LoggingService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "logging-ns",
		},
		Spec: LoggingServiceSpec{
			ContainerRuntimeType: "containerd",
		},
	}

	params := ls.ToParams()

	if params.Release.Namespace != "logging-ns" {
		t.Errorf("expected namespace logging-ns, got %q", params.Release.Namespace)
	}
	if params.Values.ContainerRuntimeType != "containerd" {
		t.Errorf("expected ContainerRuntimeType containerd, got %q", params.Values.ContainerRuntimeType)
	}
}
