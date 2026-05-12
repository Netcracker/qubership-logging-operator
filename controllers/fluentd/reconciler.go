package fluentd

import (
	"context"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	fdfactory "github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build/fluentd"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FluentdReconciler struct {
	*util.ComponentReconciler
	ComponentList     *[]util.Component
	DynamicParameters util.DynamicParameters
	Cfg               *config.Defaults
}

func NewFluentdReconciler(c client.Client, scheme *runtime.Scheme, updater util.StatusUpdater, pendingComponents *[]util.Component, dynamicParameters util.DynamicParameters, cfg *config.Defaults) FluentdReconciler {
	return FluentdReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client:        c,
			Scheme:        scheme,
			Log:           util.Logger("fluentd"),
			StatusUpdater: updater,
		},
		ComponentList:     pendingComponents,
		DynamicParameters: dynamicParameters,
		Cfg:               cfg,
	}
}

// Run reconciles ConfigMap, DaemonSet, and Service for FluentD. The DaemonSet and
// Service come from the Go factory; the ConfigMap is still rendered from embedded
// conf.d via fluentdConfigMap.
func (r *FluentdReconciler) Run(ctx context.Context, cr *loggingService.LoggingService) error {
	if !r.StatusUpdater.IsStatusFailed(util.FluentdStatus) {
		r.StatusUpdater.UpdateStatus(util.FluentdStatus, util.InProgress, false, "Start reconcile of Fluentd")
	}
	r.Log.Info("Start Fluentd reconciliation")

	if !cr.Spec.Fluentd.IsInstall() {
		r.Log.Info("Uninstalling component if exists")
		r.uninstall(cr)
		r.StatusUpdater.RemoveStatus(util.FluentdStatus)
		return nil
	}

	if err := r.handleConfigMap(cr); err != nil {
		return err
	}

	_, err := util.ReconcileAndTrackStatus(ctx, r.Client, cr, &r.StatusUpdater, util.FluentdStatus, func() (ctrl.Result, error) {
		cr.Spec.ContainerRuntimeType = r.DynamicParameters.ContainerRuntimeType
		return ctrl.Result{}, fdfactory.CreateOrUpdate(ctx, r.Client, r.Scheme, cr, r.Cfg)
	})
	if err != nil {
		return err
	}

	*r.ComponentList = append(*r.ComponentList, util.Component{
		ComponentName: util.FluentdComponentName,
		StatusName:    util.FluentdStatus,
	})
	r.Log.Info("Component reconciled")
	return nil
}

func (r *FluentdReconciler) uninstall(cr *loggingService.LoggingService) {
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
