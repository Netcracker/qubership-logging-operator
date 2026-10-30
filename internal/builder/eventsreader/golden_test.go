package eventsreader_test

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/buildertest"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/eventsreader"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
)

func TestGolden(t *testing.T) {
	cfg := defaults.Get()
	buildertest.Run(t, func(cr *loggingService.LoggingService) map[string]any {
		if cr.Spec.CloudEventsReader == nil {
			return nil
		}
		return map[string]any{
			"eventsreader-deployment": eventsreader.Deployment(cr, cfg),
			"eventsreader-service":    eventsreader.Service(cr, cfg),
		}
	})
}
