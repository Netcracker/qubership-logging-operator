package fluentbit

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

func (r *FluentbitReconciler) handleDaemonSet(cr *loggingService.LoggingService) error {
	m, err := fluentbitDaemonSet(cr, r.DynamicParameters)
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

func (r *FluentbitReconciler) handleService(cr *loggingService.LoggingService) error {
	m, err := fluentbitService(cr, r.DynamicParameters)
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

func (r *FluentbitReconciler) handleConfigSecret(cr *loggingService.LoggingService) error {
	credentials, err := r.resolveOutputCredentials(cr)
	if err != nil {
		r.Log.Error(err, "Failed to resolve Fluent Bit output credentials")
		return err
	}

	secret, err := fluentbitConfigSecret(cr, r.DynamicParameters, credentials)
	if err != nil {
		r.Log.Error(err, "Failed creating Secret manifest")
		return err
	}

	_, err = r.CreateOrUpdate(cr, secret)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Cannot create or update config secret %s", secret.Name))
		return err
	}

	return nil
}

func (r *FluentbitReconciler) resolveOutputCredentials(cr *loggingService.LoggingService) (outputCredentials, error) {
	credentials := outputCredentials{}
	if cr.Spec.Fluentbit == nil || cr.Spec.Fluentbit.Output == nil {
		return credentials, nil
	}
	namespace := cr.GetNamespace()
	output := cr.Spec.Fluentbit.Output
	if output.Loki != nil && output.Loki.Enabled && output.Loki.Auth != nil {
		values, err := r.ResolveAuthValues(namespace, output.Loki.Auth)
		if err != nil {
			return outputCredentials{}, err
		}
		credentials.Loki = values
	}
	if output.Http != nil && output.Http.Enabled && output.Http.Auth != nil {
		values, err := r.ResolveAuthValues(namespace, output.Http.Auth)
		if err != nil {
			return outputCredentials{}, err
		}
		credentials.Http = values
	}
	if output.Otel != nil && output.Otel.Enabled && output.Otel.Auth != nil {
		values, err := r.ResolveAuthValues(namespace, output.Otel.Auth)
		if err != nil {
			return outputCredentials{}, err
		}
		credentials.Otel = values
	}
	return credentials, nil
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
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	return nil
}

func (r *FluentbitReconciler) deleteConfigSecret(cr *loggingService.LoggingService) error {
	e := &corev1.Secret{
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
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	return nil
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
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	return nil
}

func (r *FluentbitReconciler) Equal(source *corev1.Secret, target *corev1.Secret) bool {
	return cmp.Equal(source.Data, target.Data) && cmp.Equal(source.GetLabels(), target.GetLabels())
}

func (r *FluentbitReconciler) CreateOrUpdate(cr *loggingService.LoggingService, secret *corev1.Secret) (created bool, err error) {
	if err = r.CreateResource(cr, secret); err != nil {
		if errors.IsAlreadyExists(err) {
			existedSecret := &corev1.Secret{ObjectMeta: secret.ObjectMeta}
			if err = r.GetResource(existedSecret); err != nil {
				return false, err
			}

			if !r.Equal(existedSecret, secret) {
				secret.SetResourceVersion(existedSecret.GetResourceVersion())
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
