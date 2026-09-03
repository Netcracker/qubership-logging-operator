package events_reader

import (
	"maps"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *EventsReaderReconciler) handleDeployment(cr *loggingService.LoggingService) error {
	desired, err := eventsReaderDeployment(cr)
	if err != nil {
		r.Log.Error(err, "Failed creating Deployment manifest")
		return err
	}

	if err = r.CreateResource(cr, desired); err == nil {
		return nil
	}
	if !errors.IsAlreadyExists(err) {
		return err
	}
	return r.updateDeployment(desired)
}

func (r *EventsReaderReconciler) updateDeployment(desired *appsv1.Deployment) error {
	existing := &appsv1.Deployment{ObjectMeta: desired.ObjectMeta}
	if err := r.GetResource(existing); err != nil {
		return err
	}

	if existing.Labels == nil && desired.Labels != nil {
		existing.SetLabels(desired.Labels)
	} else {
		maps.Copy(existing.Labels, desired.Labels)
	}
	existing.Spec.Selector = desired.Spec.Selector
	existing.Spec.Template.SetLabels(desired.Spec.Template.GetLabels())
	existing.Spec.Template.Spec.SecurityContext = desired.Spec.Template.Spec.SecurityContext
	existing.Spec.Template.Spec.Containers = desired.Spec.Template.Spec.Containers
	existing.Spec.Template.Spec.Volumes = desired.Spec.Template.Spec.Volumes
	existing.Spec.Template.Spec.ServiceAccountName = desired.Spec.Template.Spec.ServiceAccountName
	existing.Spec.Template.Spec.NodeSelector = desired.Spec.Template.Spec.NodeSelector
	existing.Spec.Template.Spec.Affinity = desired.Spec.Template.Spec.Affinity
	existing.Spec.Template.Spec.PriorityClassName = desired.Spec.Template.Spec.PriorityClassName

	return r.UpdateResource(existing)
}

func (r *EventsReaderReconciler) handleService(cr *loggingService.LoggingService) error {
	m, err := eventsReaderService(cr)
	if err != nil {
		r.Log.Error(err, "Failed creating Service manifest")
		return err
	}

	if err = r.CreateResource(cr, m); err != nil {
		if errors.IsAlreadyExists(err) {
			e := &corev1.Service{ObjectMeta: m.ObjectMeta}
			if err = r.GetResource(e); err != nil {
				return err
			}

			//Set parameters
			e.SetLabels(m.GetLabels())
			e.Spec.Ports = m.Spec.Ports
			e.Spec.Selector = m.Spec.Selector

			if err = r.UpdateResource(e); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func (r *EventsReaderReconciler) deleteDeployment(cr *loggingService.LoggingService) error {
	e := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.EventsReaderComponentName,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	return nil
}

func (r *EventsReaderReconciler) deleteService(cr *loggingService.LoggingService) error {
	e := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.EventsReaderComponentName,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	return nil
}

func (r *EventsReaderReconciler) deleteServiceAccount(cr *loggingService.LoggingService) error {
	e := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.EventsReaderComponentName,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	return nil
}
