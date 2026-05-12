package fluentbit

import (
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// handleConfigMap reconciles the FluentBit ConfigMap (the DaemonSet/Service path was
// moved to internal/reconciler/factory/build/fluentbit in Stage 2).
func (r *FluentbitReconciler) handleConfigMap(cr *loggingService.LoggingService) error {
	cm, err := fluentbitConfigMap(cr, r.DynamicParameters)
	if err != nil {
		r.Log.Error(err, "Failed creating ConfigMap manifest")
		return err
	}
	_, err = r.CreateOrUpdate(cr, cm)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Cannot create or update config map %s", cm.Name))
		return err
	}
	return nil
}

func (r *FluentbitReconciler) deleteDaemonSet(cr *loggingService.LoggingService) error {
	e := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.FluentbitComponentName,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

func (r *FluentbitReconciler) deleteConfigMap(cr *loggingService.LoggingService) error {
	e := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.FluentbitComponentName,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

func (r *FluentbitReconciler) deleteService(cr *loggingService.LoggingService) error {
	e := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.FluentbitComponentName,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

// Equal returns true if both ConfigMaps carry the same Data and BinaryData. Labels are
// ignored intentionally (covered by TestFluentbitEqual in handler_test.go).
func (r *FluentbitReconciler) Equal(source *corev1.ConfigMap, target *corev1.ConfigMap) bool {
	return cmp.Equal(source.Data, target.Data) && cmp.Equal(source.BinaryData, target.BinaryData)
}

// CreateOrUpdate creates the ConfigMap, or updates it when the live copy's Data /
// BinaryData differ from the desired state. Returns true on actual write.
func (r *FluentbitReconciler) CreateOrUpdate(cr *loggingService.LoggingService, configMap *corev1.ConfigMap) (created bool, err error) {
	if err = r.CreateResource(cr, configMap); err != nil {
		if errors.IsAlreadyExists(err) {
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
