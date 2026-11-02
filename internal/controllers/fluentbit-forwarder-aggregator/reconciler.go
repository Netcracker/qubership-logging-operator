package fluentbit_forwarder_aggregator

import (
	"context"
	"errors"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HAFluentReconciler struct {
	*util.ComponentReconciler
	ComponentList     *[]util.Component
	DynamicParameters util.DynamicParameters
	Defaults          *defaults.Defaults
}

func NewHAFluentReconciler(client client.Client, scheme *runtime.Scheme, updater util.StatusUpdater, pendingComponents *[]util.Component,
	dynamicParameters util.DynamicParameters, cfg *defaults.Defaults) HAFluentReconciler {
	return HAFluentReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client:        client,
			Scheme:        scheme,
			Log:           util.Logger("fluentbit-forwarder-aggregator"),
			StatusUpdater: updater,
		},
		ComponentList:     pendingComponents,
		DynamicParameters: dynamicParameters,
		Defaults:          cfg,
	}
}

// Run creates or updates the forwarder and aggregator objects of the FluentBit HA
// mode, or deletes them when the HA mode is not installed.
func (r *HAFluentReconciler) Run(ctx context.Context, cr *loggingService.LoggingService) error {
	if !r.StatusUpdater.IsStatusFailed(util.HAFluentStatus) {
		r.StatusUpdater.UpdateStatus(util.HAFluentStatus, util.InProgress, false, "Start reconcile of Fluentbit-Forwarder-Aggregator")
	}
	r.Log.Info("Start Fluentbit-Forwarder-Aggregator reconciliation")

	if cr.Spec.Fluentbit != nil && cr.Spec.Fluentbit.IsInstall() && cr.Spec.Fluentbit.Aggregator != nil && cr.Spec.Fluentbit.Aggregator.Install {
		if cr.Spec.Fluentbit.Aggregator.GraylogOutput && (cr.Spec.Fluentbit.Aggregator.GraylogHost == "" || cr.Spec.Fluentbit.Aggregator.GraylogPort == 0) {
			err := errors.New("configuration error: fluentbit.aggregator.graylogHost and fluentbit.aggregator.graylogPort are required with Graylog output")
			r.Log.Error(err, "configuration of fluentbit aggregator is incorrect")
			return err
		}
		if err := r.handleAggregatorConfigSecret(cr); err != nil {
			r.Log.Error(err, "error occurred in handleAggregatorConfigSecret")
			return err
		}
		if err := r.handleAggregatorStatefulSet(ctx, cr); err != nil {
			r.Log.Error(err, "error occurred in handleAggregatorStatefulSet")
			return err
		}
		if err := r.handleAggregatorService(ctx, cr); err != nil {
			r.Log.Error(err, "error occurred in handleAggregatorService")
			return err
		}

		if err := r.handleForwarderConfigMap(cr); err != nil {
			r.Log.Error(err, "error occurred in handleForwarderConfigMap")
			return err
		}
		if err := r.handleForwarderDaemonSet(ctx, cr); err != nil {
			r.Log.Error(err, "error occurred in handleForwarderDaemonSet")
			return err
		}
		if err := r.handleForwarderService(ctx, cr); err != nil {
			r.Log.Error(err, "error occurred in handleForwarderService")
			return err
		}

		*r.ComponentList = append(
			*r.ComponentList,
			util.Component{
				ComponentName: util.ForwarderFluentbitComponentName,
				StatusName:    util.HAFluentStatus,
			},
		)
	} else {
		r.Log.Info("Uninstalling component if exists")
		r.uninstall(cr)
		r.StatusUpdater.RemoveStatus(util.HAFluentStatus)
	}
	r.Log.Info("Component reconciled")
	return nil
}

// uninstall deletes all resources related to the component
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
	if err := r.deleteSecret(cr, util.AggregatorFluentbitComponentName); err != nil {
		r.Log.Error(err, "Can not delete config Secret")
	}
	if err := r.DeleteLegacyConfigMap(cr.GetNamespace(), util.AggregatorFluentbitComponentName); err != nil {
		r.Log.Error(err, "Can not delete the legacy Config Map")
	}
	if err := r.deleteService(cr, util.AggregatorFluentbitComponentName); err != nil {
		r.Log.Error(err, "Can not delete Service")
	}
}
