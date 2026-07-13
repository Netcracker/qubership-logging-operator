package fluentd

import (
	"fmt"
	"maps"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *FluentdReconciler) handleConfigSecret(cr *loggingService.LoggingService) error {
	if err := r.resolveOutputCredentials(cr); err != nil {
		r.Log.Error(err, "Failed to resolve Fluentd output credentials")
		return err
	}

	secret, err := fluentdConfigSecret(cr, r.DynamicParameters)
	if err != nil {
		r.Log.Error(err, "Failed creating Secret manifest")
		return err
	}

	_, err = r.createOrUpdateConfigSecret(cr, secret)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Cannot create or update config secret %s", secret.Name))
		return err
	}

	return nil
}

func (r *FluentdReconciler) resolveOutputCredentials(cr *loggingService.LoggingService) error {
	if cr.Spec.Fluentd == nil || cr.Spec.Fluentd.Output == nil {
		return nil
	}
	namespace := cr.GetNamespace()
	output := cr.Spec.Fluentd.Output
	if output.Loki != nil && output.Loki.Enabled {
		if err := r.ResolveAuthValues(namespace, output.Loki.Auth); err != nil {
			return err
		}
	}
	if output.Http != nil && output.Http.Enabled {
		if err := r.ResolveAuthValues(namespace, output.Http.Auth); err != nil {
			return err
		}
	}
	return nil
}

func (r *FluentdReconciler) handleDaemonSet(cr *loggingService.LoggingService) error {
	m, err := fluentdDaemonSet(cr, r.DynamicParameters)
	if err != nil {
		r.Log.Error(err, "Failed creating DaemonSet manifest")
		return err
	}

	if err = r.CreateResource(cr, m); err != nil {
		if errors.IsAlreadyExists(err) {
			e := &appsv1.DaemonSet{ObjectMeta: m.ObjectMeta}
			if err = r.GetResource(e); err != nil {
				return err
			}

			//Set parameters
			if e.Labels == nil && m.Labels != nil {
				e.SetLabels(m.Labels)
			} else {
				maps.Copy(e.Labels, m.Labels)
			}
			e.Spec.Template.SetLabels(m.Spec.Template.GetLabels())
			e.Spec.Template.Spec.Containers = m.Spec.Template.Spec.Containers
			e.Spec.Template.Spec.ServiceAccountName = m.Spec.Template.Spec.ServiceAccountName
			e.Spec.Template.Spec.NodeSelector = m.Spec.Template.Spec.NodeSelector
			e.Spec.Template.Spec.Volumes = m.Spec.Template.Spec.Volumes
			e.Spec.Template.Spec.Tolerations = m.Spec.Template.Spec.Tolerations
			e.Spec.Template.Spec.Affinity = m.Spec.Template.Spec.Affinity
			if err = r.UpdateResource(e); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func (r *FluentdReconciler) handleService(cr *loggingService.LoggingService) error {
	m, err := fluentdService(cr, r.DynamicParameters)
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

func (r *FluentdReconciler) deleteDaemonSet(cr *loggingService.LoggingService) error {
	e := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.FluentdComponentName,
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

func (r *FluentdReconciler) deleteConfigSecret(cr *loggingService.LoggingService) error {
	e := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.FluentdComponentName,
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

func (r *FluentdReconciler) deleteService(cr *loggingService.LoggingService) error {
	e := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.FluentdComponentName,
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

func (r *FluentdReconciler) Equal(source, target *corev1.Secret) bool {
	return cmp.Equal(source.Data, target.Data) &&
		cmp.Equal(source.GetLabels(), target.GetLabels())
}

func (r *FluentdReconciler) createOrUpdateConfigSecret(cr *loggingService.LoggingService, secret *corev1.Secret) (created bool, err error) {
	if err = r.CreateResource(cr, secret); err != nil {
		if errors.IsAlreadyExists(err) {
			existedSecret := &corev1.Secret{ObjectMeta: secret.ObjectMeta}
			if err = r.GetResource(existedSecret); err != nil {
				return false, err
			}

			if !r.Equal(existedSecret, secret) {
				if err = r.UpdateResource(secret); err != nil {
					return false, err
				}

				return true, nil
			}

			r.Log.Info("The config secret is not changed")
			return false, nil
		}

		return false, err
	}

	return true, nil
}
