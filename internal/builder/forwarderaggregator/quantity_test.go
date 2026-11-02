package forwarderaggregator

import (
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	appsv1 "k8s.io/api/apps/v1"
)

func mustForwarderDaemonSet(t *testing.T, cr *loggingService.LoggingService, cfg *defaults.Defaults) *appsv1.DaemonSet {
	t.Helper()
	daemonSet, err := ForwarderDaemonSet(cr, cfg)
	if err != nil {
		t.Fatalf("build the forwarder DaemonSet: %v", err)
	}
	return daemonSet
}

func mustAggregatorStatefulSet(t *testing.T, cr *loggingService.LoggingService, cfg *defaults.Defaults) *appsv1.StatefulSet {
	t.Helper()
	statefulSet, err := AggregatorStatefulSet(cr, cfg)
	if err != nil {
		t.Fatalf("build the aggregator StatefulSet: %v", err)
	}
	return statefulSet
}

// The CRD accepts any string in these fields, so a malformed quantity must fail the
// build with an error instead of a panic.
func TestMalformedQuantitiesReturnErrors(t *testing.T) {
	tests := []struct {
		name  string
		field string
		build func(cr *loggingService.LoggingService) error
	}{
		{
			name:  "forwarder memory-only state size limit",
			field: "fluentbit.memoryOnlyStateSizeLimit",
			build: func(cr *loggingService.LoggingService) error {
				cr.Spec.ContainerRuntimeType = "containerd"
				cr.Spec.Fluentbit.StorageProfile = loggingService.FluentbitStorageProfileMemoryOnly
				cr.Spec.Fluentbit.MemoryOnlyStateSizeLimit = "not-a-quantity"
				_, err := ForwarderDaemonSet(cr, defaults.Get())
				return err
			},
		},
		{
			name:  "aggregator storage size limit",
			field: "fluentbit.aggregator.storageSizeLimit",
			build: func(cr *loggingService.LoggingService) error {
				cr.Spec.Fluentbit.Aggregator.StorageSizeLimit = "not-a-quantity"
				_, err := AggregatorStatefulSet(cr, defaults.Get())
				return err
			},
		},
		{
			name:  "aggregator volume storage size",
			field: "fluentbit.aggregator.volume.storageSize",
			build: func(cr *loggingService.LoggingService) error {
				cr.Spec.Fluentbit.Aggregator.Volume = &loggingService.Volume{Bind: true, StorageSize: "not-a-quantity"}
				_, err := AggregatorStatefulSet(cr, defaults.Get())
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.build(newAggregatorSecurityLoggingService(false))
			if err == nil {
				t.Fatal("expected an error for a malformed quantity")
			}
			if !strings.Contains(err.Error(), test.field) {
				t.Errorf("error %q does not name the field %s", err, test.field)
			}
		})
	}
}
