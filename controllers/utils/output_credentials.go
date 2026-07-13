package utils

import (
	"context"
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// StringMapToByteMap converts a map of strings to Secret data.
func StringMapToByteMap(in map[string]string) map[string][]byte {
	out := make(map[string][]byte, len(in))
	for key, value := range in {
		out[key] = []byte(value)
	}
	return out
}

// ResolveSecretKeyValue reads one referenced Secret value.
func (r *ComponentReconciler) ResolveSecretKeyValue(namespace string, selector *corev1.SecretKeySelector) (string, error) {
	if selector == nil {
		return "", nil
	}
	secret := &corev1.Secret{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: selector.Name, Namespace: namespace}, secret); err != nil {
		return "", fmt.Errorf("cannot read secret %q in namespace %q: %w", selector.Name, namespace, err)
	}
	value, ok := secret.Data[selector.Key]
	if !ok || len(value) == 0 {
		return "", fmt.Errorf("key %q not found or empty in secret %q in namespace %q", selector.Key, selector.Name, namespace)
	}
	return string(value), nil
}

// ResolveAuthValues resolves Secret references into transient Auth fields.
func (r *ComponentReconciler) ResolveAuthValues(namespace string, auth *loggingService.Auth) error {
	if auth == nil {
		return nil
	}
	if auth.Token != nil {
		value, err := r.ResolveSecretKeyValue(namespace, auth.Token)
		if err != nil {
			return err
		}
		auth.TokenValue = value
	}
	if auth.User != nil {
		value, err := r.ResolveSecretKeyValue(namespace, auth.User)
		if err != nil {
			return err
		}
		auth.UserValue = value
	}
	if auth.Password != nil {
		value, err := r.ResolveSecretKeyValue(namespace, auth.Password)
		if err != nil {
			return err
		}
		auth.PasswordValue = value
	}
	return nil
}
