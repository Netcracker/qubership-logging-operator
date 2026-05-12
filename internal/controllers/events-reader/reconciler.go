package events_reader

import (
	"context"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build/eventsreader"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	appsv1 "k8s.io/api/apps/v1"
)

type EventsReaderReconciler struct {
	*util.ComponentReconciler
	ComponentList *[]util.Component
	Cfg           *config.Defaults
}

func NewEventsReaderReconciler(c client.Client, scheme *runtime.Scheme, updater util.StatusUpdater, pendingComponents *[]util.Component, cfg *config.Defaults) EventsReaderReconciler {
	return EventsReaderReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client:        c,
			Scheme:        scheme,
			Log:           util.Logger("events-reader"),
			StatusUpdater: updater,
		},
		ComponentList: pendingComponents,
		Cfg:           cfg,
	}
}

// Run reconciles the CloudEventsReader deployment + service.
func (r *EventsReaderReconciler) Run(ctx context.Context, cr *loggingService.LoggingService) error {
	if !r.StatusUpdater.IsStatusFailed(util.EventsReaderStatus) {
		r.StatusUpdater.UpdateStatus(util.EventsReaderStatus, util.InProgress, false, "Start reconcile of Events Reader")
	}
	r.Log.Info("Start Events Reader reconciliation")

	if !cr.Spec.CloudEventsReader.IsInstall() {
		r.Log.Info("Uninstalling component if exists")
		r.uninstall(cr)
		r.StatusUpdater.RemoveStatus(util.EventsReaderStatus)
		return nil
	}

	_, err := util.ReconcileAndTrackStatus(ctx, r.Client, cr, &r.StatusUpdater, util.EventsReaderStatus, func() (ctrl.Result, error) {
		return ctrl.Result{}, eventsreader.CreateOrUpdate(ctx, r.Client, r.Scheme, cr, r.Cfg)
	})
	if err != nil {
		return err
	}

	*r.ComponentList = append(*r.ComponentList, util.Component{
		ComponentName: util.EventsReaderComponentName,
		StatusName:    util.EventsReaderStatus,
	})
	r.Log.Info("Component reconciled")
	return nil
}

// uninstall removes Deployment, Service, and ServiceAccount when the CR section is absent.
func (r *EventsReaderReconciler) uninstall(cr *loggingService.LoggingService) {
	r.deleteIfExists(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: util.EventsReaderComponentName, Namespace: cr.GetNamespace()}}, "Deployment")
	r.deleteIfExists(&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: util.EventsReaderComponentName, Namespace: cr.GetNamespace()}}, "Service")
	r.deleteIfExists(&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: util.EventsReaderComponentName, Namespace: cr.GetNamespace()}}, "ServiceAccount")
}

func (r *EventsReaderReconciler) deleteIfExists(obj util.K8sResource, kind string) {
	if err := r.GetResource(obj); err != nil {
		if errors.IsNotFound(err) {
			return
		}
		r.Log.Error(err, "Get failed before delete", "kind", kind)
		return
	}
	if err := r.DeleteResource(obj); err != nil {
		r.Log.Error(err, "Delete failed", "kind", kind)
	}
}
