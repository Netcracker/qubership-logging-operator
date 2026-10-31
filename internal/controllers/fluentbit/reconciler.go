package fluentbit

import (
	"context"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FluentbitReconciler struct {
	*util.ComponentReconciler
	ComponentList     *[]util.Component
	DynamicParameters util.DynamicParameters
	Defaults          *defaults.Defaults
}

func NewFluentbitReconciler(client client.Client, scheme *runtime.Scheme, updater util.StatusUpdater,
	pendingComponents *[]util.Component, dynamicParameters util.DynamicParameters, cfg *defaults.Defaults) FluentbitReconciler {
	return FluentbitReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client:        client,
			Scheme:        scheme,
			Log:           util.Logger("fluentbit"),
			StatusUpdater: updater,
		},
		ComponentList:     pendingComponents,
		DynamicParameters: dynamicParameters,
		Defaults:          cfg,
	}
}

// Run creates or updates the FluentBit configuration Secret, DaemonSet, and Service, or
// deletes them when FluentBit is not installed or runs in the HA mode.
func (r *FluentbitReconciler) Run(ctx context.Context, cr *loggingService.LoggingService) error {
	if !r.StatusUpdater.IsStatusFailed(util.FluentbitStatus) {
		r.StatusUpdater.UpdateStatus(util.FluentbitStatus, util.InProgress, false, "Start reconcile of Fluentbit")
	}
	r.Log.Info("Start Fluentbit reconciliation")

	if cr.Spec.Fluentbit != nil && cr.Spec.Fluentbit.IsInstall() && (cr.Spec.Fluentbit.Aggregator == nil || !cr.Spec.Fluentbit.Aggregator.Install) {
		if err := r.handleConfigSecret(cr); err != nil {
			return err
		}
		if err := r.handleDaemonSet(ctx, cr); err != nil {
			return err
		}
		if err := r.handleService(ctx, cr); err != nil {
			return err
		}

		*r.ComponentList = append(
			*r.ComponentList,
			util.Component{
				ComponentName: util.FluentbitComponentName,
				StatusName:    util.FluentbitStatus,
			},
		)
	} else {
		r.Log.Info("Uninstalling component if exists")
		r.uninstall(cr)
		r.StatusUpdater.RemoveStatus(util.FluentbitStatus)
	}
	r.Log.Info("Component reconciled")
	return nil
}

// uninstall deletes all resources related to the component
func (r *FluentbitReconciler) uninstall(cr *loggingService.LoggingService) {
	if err := r.deleteDaemonSet(cr); err != nil {
		r.Log.Error(err, "Can not delete DaemonSet")
	}
	if err := r.deleteConfigSecret(cr); err != nil {
		r.Log.Error(err, "Can not delete config Secret")
	}
	if err := r.DeleteLegacyConfigMap(cr.GetNamespace(), util.FluentbitComponentName); err != nil {
		r.Log.Error(err, "Can not delete the legacy Config Map")
	}
	if err := r.deleteService(cr); err != nil {
		r.Log.Error(err, "Can not delete Service")
	}
}
