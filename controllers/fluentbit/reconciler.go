package fluentbit

import (
	"context"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	fbfactory "github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build/fluentbit"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FluentbitReconciler struct {
	*util.ComponentReconciler
	ComponentList     *[]util.Component
	DynamicParameters util.DynamicParameters
	Cfg               *config.Defaults
}

func NewFluentbitReconciler(c client.Client, scheme *runtime.Scheme, updater util.StatusUpdater, pendingComponents *[]util.Component, dynamicParameters util.DynamicParameters, cfg *config.Defaults) FluentbitReconciler {
	return FluentbitReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client:        c,
			Scheme:        scheme,
			Log:           util.Logger("fluentbit"),
			StatusUpdater: updater,
		},
		ComponentList:     pendingComponents,
		DynamicParameters: dynamicParameters,
		Cfg:               cfg,
	}
}

// Run reconciles ConfigMap, DaemonSet, and Service for FluentBit. The DaemonSet and
// Service come from the Go factory; the ConfigMap is still rendered from the embedded
// conf.d directory via fluentbitConfigMap.
func (r *FluentbitReconciler) Run(ctx context.Context, cr *loggingService.LoggingService) error {
	if !r.StatusUpdater.IsStatusFailed(util.FluentbitStatus) {
		r.StatusUpdater.UpdateStatus(util.FluentbitStatus, util.InProgress, false, "Start reconcile of Fluentbit")
	}
	r.Log.Info("Start Fluentbit reconciliation")

	if !cr.Spec.Fluentbit.IsInstall() || (cr.Spec.Fluentbit.Aggregator != nil && cr.Spec.Fluentbit.Aggregator.Install) {
		r.Log.Info("Uninstalling component if exists")
		r.uninstall(cr)
		r.StatusUpdater.RemoveStatus(util.FluentbitStatus)
		return nil
	}

	if err := r.handleConfigMap(cr); err != nil {
		return err
	}

	_, err := util.ReconcileAndTrackStatus(ctx, r.Client, cr, &r.StatusUpdater, util.FluentbitStatus, func() (ctrl.Result, error) {
		// Container runtime is dynamic; thread it onto the CR so the factory sees it.
		cr.Spec.ContainerRuntimeType = r.DynamicParameters.ContainerRuntimeType
		return ctrl.Result{}, fbfactory.CreateOrUpdate(ctx, r.Client, r.Scheme, cr, r.Cfg)
	})
	if err != nil {
		return err
	}

	*r.ComponentList = append(*r.ComponentList, util.Component{
		ComponentName: util.FluentbitComponentName,
		StatusName:    util.FluentbitStatus,
	})
	r.Log.Info("Component reconciled")
	return nil
}

func (r *FluentbitReconciler) uninstall(cr *loggingService.LoggingService) {
	if err := r.deleteDaemonSet(cr); err != nil {
		r.Log.Error(err, "Can not delete DaemonSet")
	}
	if err := r.deleteConfigMap(cr); err != nil {
		r.Log.Error(err, "Can not delete ConfigMap")
	}
	if err := r.deleteService(cr); err != nil {
		r.Log.Error(err, "Can not delete Service")
	}
}
