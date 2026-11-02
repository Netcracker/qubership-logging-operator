package graylog_test

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/buildertest"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/graylog"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
)

func TestGolden(t *testing.T) {
	cfg := defaults.Get()
	buildertest.Run(t, func(cr *loggingService.LoggingService) map[string]any {
		if cr.Spec.Graylog == nil {
			return nil
		}
		return map[string]any{
			"graylog-statefulset":    graylog.StatefulSet(cr, cfg),
			"graylog-service":        graylog.Service(cr, cfg),
			"graylog-serviceaccount": graylog.ServiceAccount(cr, cfg),
		}
	})
}
