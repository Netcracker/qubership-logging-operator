package utils

import (
	"context"
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HandleReconcileErrWithStatus is the thin wrapper component reconcilers use to route
// any error returned from their CreateOrUpdate-style logic through the existing
// StatusUpdater and decide on a ctrl.Result. Concrete on *loggingService.LoggingService
// for now — generalize only if a second CRD is introduced.
//
// Behavior:
//   - err == nil: pass through originResult, nil.
//   - err != nil: record a Failed condition for reason on the CR via updater, then
//     return originResult and the same err so controller-runtime requeues.
//
// This is intentionally lighter than the VM-operator port (no Event emission, no
// parsingError discrimination, no conflict-retry); we'll layer those in only if
// concrete needs surface during component migrations.
func HandleReconcileErrWithStatus(_ context.Context, _ client.Client, _ *loggingService.LoggingService, updater *StatusUpdater, reason string, originResult ctrl.Result, err error) (ctrl.Result, error) {
	if err == nil {
		return originResult, nil
	}
	if updater != nil {
		updater.UpdateStatus(reason, Failed, false, fmt.Sprintf("Reason: %s", err.Error()))
	}
	return originResult, err
}

// ReconcileAndTrackStatus runs cb and records an InProgress->Success transition on the
// CR via updater. Errors from cb are returned untouched so the caller can route them
// through HandleReconcileErrWithStatus. Mirrors the VM-operator's reconcileAndTrackStatus
// in spirit but stays simple — no spec-change detection, no Paused() handling.
func ReconcileAndTrackStatus(_ context.Context, _ client.Client, _ *loggingService.LoggingService, updater *StatusUpdater, reason string, cb func() (ctrl.Result, error)) (ctrl.Result, error) {
	if updater != nil {
		updater.UpdateStatus(reason, InProgress, false, fmt.Sprintf("%s reconcile in progress", reason))
	}
	result, err := cb()
	if err != nil {
		return result, err
	}
	if updater != nil {
		updater.UpdateStatus(reason, Success, true, fmt.Sprintf("%s reconcile succeeded", reason))
	}
	return result, nil
}
