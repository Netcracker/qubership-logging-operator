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
		daemonSet, err := forwarderaggregator.ForwarderDaemonSet(cr, cfg)
		if err != nil {
			t.Fatalf("build the forwarder DaemonSet: %v", err)
		}
		statefulSet, err := forwarderaggregator.AggregatorStatefulSet(cr, cfg)
		if err != nil {
			t.Fatalf("build the aggregator StatefulSet: %v", err)
		}
		return map[string]any{
			"forwarder-daemonset":    daemonSet,
			"forwarder-service":      forwarderaggregator.ForwarderService(cr, cfg),
			"aggregator-statefulset": statefulSet,
			"aggregator-service":     forwarderaggregator.AggregatorService(cr, cfg),
		}
	})
}
