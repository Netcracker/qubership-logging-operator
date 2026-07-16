package utils

import (
	"context"
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// AuthValues contains credentials resolved for configuration rendering.
type AuthValues struct {
	Token    string
	User     string
	Password string
}

// StringMapToByteMap converts a map of string values into a map of byte slices,
// suitable for the Data field of a corev1.Secret.
func StringMapToByteMap(in map[string]string) map[string][]byte {
	out := make(map[string][]byte, len(in))
	for key, value := range in {
		out[key] = []byte(value)
	}
	return out
}

// ResolveSecretKeyValue reads a single key from a Secret in the given namespace
// and returns its value as a string. It is used to inline sensitive output
// credentials into the generated Fluent Bit configuration Secret instead of
// exposing them through environment variables.
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

// ResolveAuthValues resolves the Secrets referenced by an Auth block and returns
// their plain values so they can be inlined into the generated Fluent Bit
// configuration Secret instead of being exposed as environment variables. Nil
// auth or nil selectors are skipped. The resolved values are never logged.
func (r *ComponentReconciler) ResolveAuthValues(namespace string, auth *loggingService.Auth) (AuthValues, error) {
	if auth == nil {
		return AuthValues{}, nil
	}

	values := AuthValues{}
	var err error

	if auth.Token != nil {
		values.Token, err = r.ResolveSecretKeyValue(namespace, auth.Token)
		if err != nil {
			return AuthValues{}, err
		}
	}
	if auth.User != nil {
		values.User, err = r.ResolveSecretKeyValue(namespace, auth.User)
		if err != nil {
			return AuthValues{}, err
		}
	}
	if auth.Password != nil {
		values.Password, err = r.ResolveSecretKeyValue(namespace, auth.Password)
		if err != nil {
			return AuthValues{}, err
		}
	}
	return values, nil
}
