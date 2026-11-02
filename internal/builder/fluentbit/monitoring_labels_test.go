package fluentbit

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestFluentbitPodsCarryNameLabel guards the label the PodMonitor "fluentbit-all-pods"
// selects on (charts/qubership-logging-operator/templates/fluentbit-daemonset/podmonitor.yaml).
func TestFluentbitPodsCarryNameLabel(t *testing.T) {
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{Fluentbit: &loggingService.Fluentbit{
			ContainerLogging: true,
			ConfigmapReload:  &loggingService.ConfigmapReload{},
		}},
	}
	daemonSet := DaemonSet(withContainerd(cr), defaults.Get())

	if got := daemonSet.Spec.Template.Labels["app.kubernetes.io/name"]; got != util.FluentbitComponentName {
		t.Errorf("expected the pod label app.kubernetes.io/name=%s, got %q", util.FluentbitComponentName, got)
	}
}
