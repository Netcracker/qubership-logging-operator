package fluentd

import (
	"context"
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/fluentd"
	"github.com/Netcracker/qubership-logging-operator/internal/reconcilers"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
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

// handleDaemonSet creates or updates the FluentD DaemonSet for the detected container runtime.
func (r *FluentdReconciler) handleDaemonSet(ctx context.Context, cr *loggingService.LoggingService) error {
	cr.Spec.ContainerRuntimeType = r.DynamicParameters.ContainerRuntimeType
	return reconcilers.DaemonSet(ctx, r.Client, r.Scheme, cr, fluentd.DaemonSet(cr, r.Defaults))
}

func (r *FluentdReconciler) handleService(ctx context.Context, cr *loggingService.LoggingService) error {
	return reconcilers.Service(ctx, r.Client, r.Scheme, cr, fluentd.Service(cr, r.Defaults))
}

func (r *FluentdReconciler) deleteDaemonSet(cr *loggingService.LoggingService) error {
	return r.deleteIfExists(&appsv1.DaemonSet{ObjectMeta: componentMeta(cr)})
}

func (r *FluentdReconciler) deleteConfigSecret(cr *loggingService.LoggingService) error {
	if err := r.deleteIfExists(&corev1.Secret{ObjectMeta: componentMeta(cr)}); err != nil {
		return err
	}
	return r.DeleteLegacyConfigMap(cr.GetNamespace(), util.FluentdComponentName)
}

func (r *FluentdReconciler) deleteService(cr *loggingService.LoggingService) error {
	return r.deleteIfExists(&corev1.Service{ObjectMeta: componentMeta(cr)})
}

func componentMeta(cr *loggingService.LoggingService) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: util.FluentdComponentName, Namespace: cr.GetNamespace()}
}

func (r *FluentdReconciler) deleteIfExists(obj util.K8sResource) error {
	if err := r.GetResource(obj); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(obj)
}
