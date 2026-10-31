package graylog

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
)

// TestGraylogServiceCarriesNameLabel guards the label the ServiceMonitor "graylog-service-monitor"
// selects on (charts/qubership-logging-operator/templates/graylog/servicemonitor.yaml).
func TestGraylogServiceCarriesNameLabel(t *testing.T) {
	cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
		Graylog: &loggingService.Graylog{AuthProxy: &loggingService.AuthProxy{}},
	}}

	service := Service(cr, defaults.Get())

	if got := service.Labels["app.kubernetes.io/name"]; got != util.GraylogComponentName {
		t.Errorf("expected the service label app.kubernetes.io/name=%s, got %q", util.GraylogComponentName, got)
	}
}
