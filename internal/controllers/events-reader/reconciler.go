package events_reader

import (
	"context"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/eventsreader"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	"github.com/Netcracker/qubership-logging-operator/internal/reconcilers"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EventsReaderReconciler struct {
	*util.ComponentReconciler
	ComponentList *[]util.Component
	Defaults      *defaults.Defaults
}

func NewEventsReaderReconciler(client client.Client, scheme *runtime.Scheme, updater util.StatusUpdater,
	pendingComponents *[]util.Component, cfg *defaults.Defaults) EventsReaderReconciler {
	return EventsReaderReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client:        client,
			Scheme:        scheme,
			Log:           util.Logger("events-reader"),
			StatusUpdater: updater,
		},
		ComponentList: pendingComponents,
		Defaults:      cfg,
	}
}

// Run creates or updates the Events Reader Deployment and Service, or deletes them
// when the component is not installed.
func (r *EventsReaderReconciler) Run(ctx context.Context, cr *loggingService.LoggingService) error {
	if !r.StatusUpdater.IsStatusFailed(util.EventsReaderStatus) {
		r.StatusUpdater.UpdateStatus(util.EventsReaderStatus, util.InProgress, false, "Start reconcile of Events Reader")
	}
	r.Log.Info("Start Events Reader reconciliation")

	if cr.Spec.CloudEventsReader != nil && cr.Spec.CloudEventsReader.IsInstall() {
		if err := r.handleDeployment(ctx, cr); err != nil {
			return err
		}
		if err := reconcilers.Service(ctx, r.Client, r.Scheme, cr, eventsreader.Service(cr, r.Defaults)); err != nil {
			return err
		}

		*r.ComponentList = append(
			*r.ComponentList,
			util.Component{
				ComponentName: util.EventsReaderComponentName,
				StatusName:    util.EventsReaderStatus,
			},
		)
	} else {
		r.Log.Info("Uninstalling component if exists")
		r.uninstall(cr)
		r.StatusUpdater.RemoveStatus(util.EventsReaderStatus)
	}
	r.Log.Info("Component reconciled")
	return nil
}

func (r *EventsReaderReconciler) handleDeployment(ctx context.Context, cr *loggingService.LoggingService) error {
	return reconcilers.Deployment(ctx, r.Client, r.Scheme, cr, eventsreader.Deployment(cr, r.Defaults))
}

// uninstall deletes all resources related to the component
func (r *EventsReaderReconciler) uninstall(cr *loggingService.LoggingService) {
	meta := metav1.ObjectMeta{Name: util.EventsReaderComponentName, Namespace: cr.GetNamespace()}
	if err := r.deleteIfExists(&appsv1.Deployment{ObjectMeta: meta}); err != nil {
		r.Log.Error(err, "Can not delete Deployment")
	}
	if err := r.deleteIfExists(&corev1.Service{ObjectMeta: meta}); err != nil {
		r.Log.Error(err, "Can not delete Service")
	}
	if err := r.deleteIfExists(&corev1.ServiceAccount{ObjectMeta: meta}); err != nil {
		r.Log.Error(err, "Can not delete ServiceAccount")
	}
}

func (r *EventsReaderReconciler) deleteIfExists(obj util.K8sResource) error {
	if err := r.GetResource(obj); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(obj)
}
