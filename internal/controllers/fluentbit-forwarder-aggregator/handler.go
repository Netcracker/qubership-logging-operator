package fluentbit_forwarder_aggregator

import (
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// handleForwarderConfigMap reconciles the forwarder ConfigMap (DaemonSet+Service moved
// to the Go factory in Stage 4).
func (r *HAFluentReconciler) handleForwarderConfigMap(cr *loggingService.LoggingService) error {
	m, err := forwarderConfigMap(cr, r.DynamicParameters)
	if err != nil {
		r.Log.Error(err, "Failed creating ConfigMap manifest")
		return err
	}
	_, err = r.updateConfigMap(cr, m)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Cannot create or update config map %s", m.Name))
		return err
	}
	return nil
}

// handleAggregatorConfigMap reconciles the aggregator ConfigMap (StatefulSet+Service
// moved to the Go factory in Stage 4).
func (r *HAFluentReconciler) handleAggregatorConfigMap(cr *loggingService.LoggingService) error {
	m, err := aggregatorConfigMap(cr, r.DynamicParameters)
	if err != nil {
		r.Log.Error(err, "Failed creating ConfigMap manifest")
		return err
	}
	_, err = r.updateConfigMap(cr, m)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Cannot create or update config map %s", m.Name))
		return err
	}
	return nil
}

func (r *HAFluentReconciler) deleteDaemonSet(cr *loggingService.LoggingService, name string) error {
	e := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cr.GetNamespace()}}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

func (r *HAFluentReconciler) deleteStatefulSet(cr *loggingService.LoggingService, name string) error {
	e := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cr.GetNamespace()}}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

func (r *HAFluentReconciler) deleteConfigMap(cr *loggingService.LoggingService, name string) error {
	e := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cr.GetNamespace()}}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

func (r *HAFluentReconciler) deleteService(cr *loggingService.LoggingService, name string) error {
	e := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cr.GetNamespace()}}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

// Equal returns true when source and target ConfigMaps carry the same Data,
// BinaryData, and Labels. HA fluent intentionally checks labels — covered by
// TestHAFluentEqual.
func (r *HAFluentReconciler) Equal(source *corev1.ConfigMap, target *corev1.ConfigMap) bool {
	return cmp.Equal(source.Data, target.Data) &&
		cmp.Equal(source.BinaryData, target.BinaryData) &&
		cmp.Equal(source.GetLabels(), target.GetLabels())
}

// updateConfigMap creates the ConfigMap, or updates it when the live copy's Data /
// BinaryData / Labels differ from the desired state.
func (r *HAFluentReconciler) updateConfigMap(cr *loggingService.LoggingService, configMap *corev1.ConfigMap) (updated bool, err error) {
	if err = r.CreateResource(cr, configMap); err != nil {
		if api_errors.IsAlreadyExists(err) {
			existedConfigMap := &corev1.ConfigMap{ObjectMeta: configMap.ObjectMeta}
			if err = r.GetResource(existedConfigMap); err != nil {
				return false, err
			}
			if !r.Equal(existedConfigMap, configMap) {
				if err = r.UpdateResource(configMap); err != nil {
					return false, err
				}
				return true, nil
			}
			r.Log.Info("The config map is not changed")
			return false, nil
		}
		return false, err
	}
	return true, nil
}
