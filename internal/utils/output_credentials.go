package utils

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// OutputAuth binds the Auth block of a single output to the destination that
// receives its resolved credentials.
type OutputAuth struct {
	Values  *AuthValues
	Enabled bool
	Auth    *loggingService.Auth
}

// credentialValidator reports whether a resolved credential can be rendered
// verbatim into a component configuration.
type credentialValidator func(value string) error

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
// credentials into the generated configuration Secret instead of exposing them
// through environment variables.
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

// ResolveFluentdOutputAuth resolves the Auth blocks of the enabled FluentD outputs
// into their destinations. Values that FluentD cannot carry on a single line are
// rejected instead of being rendered.
func (r *ComponentReconciler) ResolveFluentdOutputAuth(namespace string, outputs ...OutputAuth) error {
	return r.resolveOutputAuth(namespace, validateSingleLineValue, outputs)
}

// ResolveFluentbitOutputAuth resolves the Auth blocks of the enabled Fluent Bit
// outputs into their destinations. Values that the classic Fluent Bit configuration
// format cannot carry verbatim are rejected instead of being rendered.
func (r *ComponentReconciler) ResolveFluentbitOutputAuth(namespace string, outputs ...OutputAuth) error {
	return r.resolveOutputAuth(namespace, validateFluentbitValue, outputs)
}

// resolveOutputAuth reads the Secrets referenced by every enabled output and stores
// their plain values in the output destination, so that they can be inlined into the
// generated configuration Secret instead of being exposed as environment variables.
// Disabled outputs, nil auth, and nil selectors are skipped. Neither the resolved
// values nor the returned errors contain the credentials themselves.
func (r *ComponentReconciler) resolveOutputAuth(namespace string, validate credentialValidator, outputs []OutputAuth) error {
	for _, output := range outputs {
		if !output.Enabled || output.Auth == nil || output.Values == nil {
			continue
		}
		values, err := r.resolveAuthValues(namespace, output.Auth, validate)
		if err != nil {
			return err
		}
		*output.Values = values
	}
	return nil
}

func (r *ComponentReconciler) resolveAuthValues(namespace string, auth *loggingService.Auth, validate credentialValidator) (AuthValues, error) {
	values := AuthValues{}
	fields := []struct {
		selector *corev1.SecretKeySelector
		target   *string
	}{
		{auth.Token, &values.Token},
		{auth.User, &values.User},
		{auth.Password, &values.Password},
	}
	for _, field := range fields {
		if field.selector == nil {
			continue
		}
		value, err := r.ResolveSecretKeyValue(namespace, field.selector)
		if err != nil {
			return AuthValues{}, err
		}
		if err := validate(value); err != nil {
			return AuthValues{}, fmt.Errorf("key %q in secret %q in namespace %q %w",
				field.selector.Key, field.selector.Name, namespace, err)
		}
		*field.target = value
	}
	return values, nil
}

// validateSingleLineValue rejects credentials that cannot be rendered on one
// configuration line. FluentD and Fluent Bit both parse their configuration line by
// line, so a line break inside a credential ends the directive and turns the rest of
// the value into configuration of its own.
func validateSingleLineValue(value string) error {
	if strings.ContainsAny(value, "\r\n\x00") {
		return errors.New("must not contain a line break or a NUL byte")
	}
	return nil
}

// validateFluentbitValue rejects credentials that the classic Fluent Bit
// configuration format cannot carry verbatim. On top of the line-break rule, Fluent
// Bit substitutes ${...} in a directive value with an environment variable, which
// would silently replace part of the credential with an empty string.
func validateFluentbitValue(value string) error {
	if err := validateSingleLineValue(value); err != nil {
		return err
	}
	if strings.Contains(value, "${") {
		return errors.New(`must not contain "${" because Fluent Bit expands it as an environment variable reference`)
	}
	return nil
}

// FluentdQuote renders a value as a single-quoted FluentD string literal.
//
// FluentD evaluates embedded Ruby expressions (#{...}) inside double-quoted
// strings, so a credential containing that sequence would either break the
// configuration or run arbitrary code in the FluentD process. Single-quoted
// literals are taken verbatim and recognize only the \' and \\ escapes.
func FluentdQuote(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return "'" + escaped + "'"
}
