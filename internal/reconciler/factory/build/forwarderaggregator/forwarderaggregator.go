// Package forwarderaggregator builds the FluentBit HA pair: the forwarder DaemonSet +
// Service (per-node log collection) and the aggregator StatefulSet + Service (central
// processing + output). Replaces the four YAML assets under
// controllers/fluentbit-forwarder-aggregator/assets/. ConfigMap construction stays in
// controllers/fluentbit-forwarder-aggregator/manifest.go.
package forwarderaggregator

import (
	"context"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/reconcile"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ForwarderName is the forwarder DaemonSet / container / Service name.
	// Matches util.ForwarderFluentbitComponentName.
	ForwarderName = "logging-fluentbit-forwarder"
	// AggregatorName is the aggregator StatefulSet / container / Service / SA name.
	// Matches util.AggregatorFluentbitComponentName.
	AggregatorName = "logging-fluentbit-aggregator"
	// ReloadContainer is the configmap-reload sidecar name in both pods.
	ReloadContainer = "configmap-reload"
	// Technology is the value of app.kubernetes.io/technology on both pod templates.
	Technology = "cpp"

	// ForwarderServiceAccount is the SA both the non-HA fluentbit DaemonSet and the
	// HA forwarder DaemonSet share. Asset-baked, preserved.
	ForwarderServiceAccount = "logging-fluentbit"

	// envCloudURL is unused here; declared for parity with the events-reader build.
	containerRuntimeDocker = "docker"
	osKindUbuntu           = "ubuntu"

	configMapDefaultMode = int32(420) // 0644
	authTokenDefaultMode = int32(220) // preserved from legacy asset
)

// CreateOrUpdate reconciles forwarder DaemonSet + Service and aggregator StatefulSet +
// Service. Caller gates on cr.Spec.Fluentbit.Aggregator.Install.
func CreateOrUpdate(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService, cfg *config.Defaults) error {
	if err := reconcile.StatefulSet(ctx, c, scheme, cr, buildAggregatorStatefulSet(cr, cfg.Aggregator)); err != nil {
		return err
	}
	if err := reconcile.Service(ctx, c, scheme, cr, buildAggregatorService(cr)); err != nil {
		return err
	}
	if err := reconcile.DaemonSet(ctx, c, scheme, cr, buildForwarderDaemonSet(cr, cfg.Aggregator)); err != nil {
		return err
	}
	return reconcile.Service(ctx, c, scheme, cr, buildForwarderService(cr))
}

func nodeSelector(key, value string) map[string]string {
	if key == "" || value == "" {
		return nil
	}
	return map[string]string{key: value}
}

func boolPtr(b bool) *bool { return &b }

func int64Ptr(i int64) *int64 { return &i }

func providerLabel(openshift bool) string {
	if openshift {
		return "openshift"
	}
	return "kubernetes"
}

// resourceProviderLabels are the bare "component" + "provider" pair the legacy assets
// wrote alongside the operator-standard labels (added later via util.SetLabels*).
func resourceProviderLabels(component string, openshift bool) map[string]string {
	return map[string]string{
		"component": component,
		"provider":  providerLabel(openshift),
	}
}

// clusterServicePodLabel adds the historical kubernetes.io/cluster-service=true marker
// that both legacy DS and STS pod templates baked in.
func clusterServicePodLabel(base map[string]string) map[string]string {
	out := make(map[string]string, len(base)+2)
	for k, v := range base {
		out[k] = v
	}
	out["app.kubernetes.io/technology"] = Technology
	out["kubernetes.io/cluster-service"] = "true"
	return out
}

// hostPathType returns a pointer to t. corev1.HostPathVolumeSource.Type is *HostPathType.
func hostPathType(t corev1.HostPathType) *corev1.HostPathType { return &t }
