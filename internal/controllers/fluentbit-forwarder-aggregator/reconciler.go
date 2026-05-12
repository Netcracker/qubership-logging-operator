package fluentbit_forwarder_aggregator

import (
	"context"
	"errors"
	"time"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build/forwarderaggregator"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HAFluentReconciler struct {
	*util.ComponentReconciler
	ComponentList     *[]util.Component
	DynamicParameters util.DynamicParameters
	Cfg               *config.Defaults
}

func NewHAFluentReconciler(c client.Client, scheme *runtime.Scheme, updater util.StatusUpdater, pendingComponents *[]util.Component, dynamicParameters util.DynamicParameters, cfg *config.Defaults) HAFluentReconciler {
	return HAFluentReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client:        c,
			Scheme:        scheme,
			Log:           util.Logger("fluentbit-forwarder-aggregator"),
			StatusUpdater: updater,
		},
		ComponentList:     pendingComponents,
		DynamicParameters: dynamicParameters,
		Cfg:               cfg,
	}
}

// Run reconciles forwarder + aggregator. ConfigMaps are still rendered from embedded
// conf.d directories; DaemonSet/StatefulSet/Services come from the Go factory.
func (r *HAFluentReconciler) Run(ctx context.Context, cr *loggingService.LoggingService) error {
	if !r.StatusUpdater.IsStatusFailed(util.HAFluentStatus) {
		r.StatusUpdater.UpdateStatus(util.HAFluentStatus, util.InProgress, false, "Start reconcile of Fluentbit-Forwarder-Aggregator")
	}
	r.Log.Info("Start Fluentbit-Forwarder-Aggregator reconciliation")

	if cr.Spec.Fluentbit == nil || !cr.Spec.Fluentbit.IsInstall() || cr.Spec.Fluentbit.Aggregator == nil || !cr.Spec.Fluentbit.Aggregator.Install {
		r.Log.Info("Uninstalling component if exists")
		r.uninstall(cr)
		r.StatusUpdater.RemoveStatus(util.HAFluentStatus)
		return nil
	}

	if cr.Spec.Fluentbit.Aggregator.GraylogOutput && (cr.Spec.Fluentbit.Aggregator.GraylogHost == "" || cr.Spec.Fluentbit.Aggregator.GraylogPort == 0) {
		err := errors.New("configuration error: fluentbit.aggregator.graylogHost and fluentbit.aggregator.graylogPort are required with Graylog output")
		r.Log.Error(err, "configuration of fluentbit aggregator is incorrect")
		return err
	}

	if err := r.handleAggregatorConfigMap(cr); err != nil {
		return err
	}
	if err := r.handleForwarderConfigMap(cr); err != nil {
		return err
	}

	_, err := util.ReconcileAndTrackStatus(ctx, r.Client, cr, &r.StatusUpdater, util.HAFluentStatus, func() (ctrl.Result, error) {
		cr.Spec.ContainerRuntimeType = r.DynamicParameters.ContainerRuntimeType
		return ctrl.Result{}, forwarderaggregator.CreateOrUpdate(ctx, r.Client, r.Scheme, cr, r.Cfg)
	})
	if err != nil {
		return err
	}

	// Preserve the legacy post-apply readiness wait for the aggregator StatefulSet.
	time.Sleep(util.InitialDelay)
	podManager := util.NewPodManager(r.Client, cr.GetNamespace(), r.Log)
	timeout := util.FluentbitAggregatorPendingTimeout
	if cr.Spec.Fluentbit.Aggregator.StartupTimeout != 0 {
		timeout = time.Duration(cr.Spec.Fluentbit.Aggregator.StartupTimeout) * time.Minute
	}
	started, err := podManager.WaitForStatefulsetUpdated(util.AggregatorFluentbitComponentName, timeout)
	if err != nil {
		return err
	}
	if !started {
		r.StatusUpdater.UpdateStatus(util.HAFluentStatus, util.Failed, false, "Fluent bit aggregator is not started")
		return errors.New("fluent bit aggregator is not started")
	}

	*r.ComponentList = append(*r.ComponentList, util.Component{
		ComponentName: util.ForwarderFluentbitComponentName,
		StatusName:    util.HAFluentStatus,
	})
	r.Log.Info("Component reconciled")
	return nil
}

func (r *HAFluentReconciler) uninstall(cr *loggingService.LoggingService) {
	if err := r.deleteDaemonSet(cr, util.ForwarderFluentbitComponentName); err != nil {
		r.Log.Error(err, "Can not delete Daemon Set")
	}
	if err := r.deleteConfigMap(cr, util.ForwarderFluentbitComponentName); err != nil {
		r.Log.Error(err, "Can not delete Config Map")
	}
	if err := r.deleteService(cr, util.ForwarderFluentbitComponentName); err != nil {
		r.Log.Error(err, "Can not delete Service")
	}
	if err := r.deleteStatefulSet(cr, util.AggregatorFluentbitComponentName); err != nil {
		r.Log.Error(err, "Can not delete Stateful Set")
	}
	if err := r.deleteConfigMap(cr, util.AggregatorFluentbitComponentName); err != nil {
		r.Log.Error(err, "Can not delete Config Map")
	}
	if err := r.deleteService(cr, util.AggregatorFluentbitComponentName); err != nil {
		r.Log.Error(err, "Can not delete Service")
	}
}
