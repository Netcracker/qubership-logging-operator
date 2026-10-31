package forwarderaggregator

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
)

// TestHAFluentbitPodsCarryNameLabel guards the labels the PodMonitor "fluentbit-all-pods"
// selects on (charts/qubership-logging-operator/templates/fluentbit-daemonset/podmonitor.yaml).
func TestHAFluentbitPodsCarryNameLabel(t *testing.T) {
	cr := &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
		Fluentbit: &loggingService.Fluentbit{
			ContainerLogging: true,
			ConfigmapReload:  &loggingService.ConfigmapReload{},
			Aggregator: &loggingService.FluentbitAggregator{
				Install:         true,
				ConfigmapReload: &loggingService.ConfigmapReload{},
				Resources:       &corev1.ResourceRequirements{},
			},
		},
	}}

	t.Run("forwarder", func(t *testing.T) {
		cr.Spec.ContainerRuntimeType = "containerd"
		ds := ForwarderDaemonSet(cr, defaults.Get())
		if got := ds.Spec.Template.Labels["app.kubernetes.io/name"]; got != util.ForwarderFluentbitComponentName {
			t.Errorf("expected the pod label app.kubernetes.io/name=%s, got %q", util.ForwarderFluentbitComponentName, got)
		}
	})

	t.Run("aggregator", func(t *testing.T) {
		statefulSet := AggregatorStatefulSet(cr, defaults.Get())
		if got := statefulSet.Spec.Template.Labels["app.kubernetes.io/name"]; got != util.AggregatorFluentbitComponentName {
			t.Errorf("expected the pod label app.kubernetes.io/name=%s, got %q", util.AggregatorFluentbitComponentName, got)
		}
	})
}
