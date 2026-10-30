package fluentbit_test

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/buildertest"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/fluentbit"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
)

func TestGolden(t *testing.T) {
	cfg := defaults.Get()
	buildertest.Run(t, func(cr *loggingService.LoggingService) map[string]any {
		if cr.Spec.Fluentbit == nil {
			return nil
		}
		return map[string]any{
			"fluentbit-daemonset": fluentbit.DaemonSet(cr, cfg),
			"fluentbit-service":   fluentbit.Service(cr, cfg),
		}
	})
}
