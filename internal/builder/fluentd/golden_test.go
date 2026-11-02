package fluentd_test

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/buildertest"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/fluentd"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
)

func TestGolden(t *testing.T) {
	cfg := defaults.Get()
	buildertest.Run(t, func(cr *loggingService.LoggingService) map[string]any {
		if cr.Spec.Fluentd == nil {
			return nil
		}
		return map[string]any{
			"fluentd-daemonset": fluentd.DaemonSet(cr, cfg),
			"fluentd-service":   fluentd.Service(cr, cfg),
		}
	})
}
