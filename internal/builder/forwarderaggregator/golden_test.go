package forwarderaggregator_test

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/buildertest"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/forwarderaggregator"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
)

func TestGolden(t *testing.T) {
	cfg := defaults.Get()
	buildertest.Run(t, func(cr *loggingService.LoggingService) map[string]any {
		if cr.Spec.Fluentbit == nil || cr.Spec.Fluentbit.Aggregator == nil {
			return nil
		}
		return map[string]any{
			"forwarder-daemonset":    forwarderaggregator.ForwarderDaemonSet(cr, cfg),
			"forwarder-service":      forwarderaggregator.ForwarderService(cr, cfg),
			"aggregator-statefulset": forwarderaggregator.AggregatorStatefulSet(cr, cfg),
			"aggregator-service":     forwarderaggregator.AggregatorService(cr, cfg),
		}
	})
}
