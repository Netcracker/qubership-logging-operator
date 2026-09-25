package utils

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EqualConfigSecret reports whether two generated configuration Secrets carry the
// same rendered configuration and the same labels.
func EqualConfigSecret(source, target *corev1.Secret) bool {
	return cmp.Equal(source.Data, target.Data) &&
		cmp.Equal(source.GetLabels(), target.GetLabels())
}

// CreateOrUpdateConfigSecret creates the generated configuration Secret, or updates
// the existing one when its rendered configuration or labels differ. It reports
// whether the Secret in the cluster changed.
//
// The desired Secret is built from scratch on every reconcile and therefore carries
// no ResourceVersion, so the version of the existing object is copied over before the
// update to satisfy optimistic concurrency.
func (r *ComponentReconciler) CreateOrUpdateConfigSecret(cr *loggingService.LoggingService, secret *corev1.Secret) (bool, error) {
	err := r.CreateResource(cr, secret)
	if err == nil {
		return true, nil
	}
	if !errors.IsAlreadyExists(err) {
		return false, err
	}

	existingSecret := &corev1.Secret{ObjectMeta: secret.ObjectMeta}
	if err := r.GetResource(existingSecret); err != nil {
		return false, err
	}
	if EqualConfigSecret(existingSecret, secret) {
		r.Log.Info("The config secret is not changed")
		return false, nil
	}

	secret.SetResourceVersion(existingSecret.GetResourceVersion())
	if err := r.UpdateResource(secret); err != nil {
		return false, err
	}
	return true, nil
}

// DeleteLegacyConfigMap removes the ConfigMap that stored a component configuration
// before it moved into the configuration Secret. Releases upgraded from those
// versions would otherwise keep an orphaned copy of the old configuration in the
// namespace, and uninstall would leave it behind for good.
func (r *ComponentReconciler) DeleteLegacyConfigMap(namespace, name string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := r.GetResource(configMap); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(configMap)
}
