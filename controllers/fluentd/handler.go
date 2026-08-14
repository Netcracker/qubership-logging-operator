package fluentd

import (
	"fmt"
	"maps"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *FluentdReconciler) handleConfigSecret(cr *loggingService.LoggingService) error {
	credentials, err := r.resolveOutputCredentials(cr)
	if err != nil {
		r.Log.Error(err, "Failed to resolve Fluentd output credentials")
		return err
	}

	secret, err := fluentdConfigSecret(cr, r.DynamicParameters, credentials)
	if err != nil {
		r.Log.Error(err, "Failed creating Secret manifest")
		return err
	}

	_, err = r.CreateOrUpdateConfigSecret(cr, secret)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Cannot create or update config secret %s", secret.Name))
		return err
	}

	if err = r.DeleteLegacyConfigMap(cr.GetNamespace(), util.FluentdComponentName); err != nil {
		r.Log.Error(err, fmt.Sprintf("Cannot delete the legacy config map %s", util.FluentdComponentName))
		return err
	}

	return nil
}

func (r *FluentdReconciler) resolveOutputCredentials(cr *loggingService.LoggingService) (outputCredentials, error) {
	credentials := outputCredentials{}
	if cr.Spec.Fluentd == nil || cr.Spec.Fluentd.Output == nil {
		return credentials, nil
	}
	output := cr.Spec.Fluentd.Output
	outputs := make([]util.OutputAuth, 0, 2)
	if output.Loki != nil {
		outputs = append(outputs, util.OutputAuth{Values: &credentials.Loki, Enabled: output.Loki.Enabled, Auth: output.Loki.Auth})
	}
	if output.Http != nil {
		outputs = append(outputs, util.OutputAuth{Values: &credentials.Http, Enabled: output.Http.Enabled, Auth: output.Http.Auth})
	}
	if err := r.ResolveFluentdOutputAuth(cr.GetNamespace(), outputs...); err != nil {
		return outputCredentials{}, err
	}
	return credentials, nil
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
			e.Spec.Template.Spec = m.Spec.Template.Spec
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
	return r.DeleteLegacyConfigMap(cr.GetNamespace(), util.FluentdComponentName)
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
