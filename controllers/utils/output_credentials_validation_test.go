package utils

import (
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newCredentialReconciler(t *testing.T, data map[string][]byte) *ComponentReconciler {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "output-auth", Namespace: "logging"},
		Data:       data,
	}
	return &ComponentReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build(),
		Scheme: scheme,
		Log:    Logger("test-credentials"),
	}
}

func passwordAuth(key string) *loggingService.Auth {
	return &loggingService.Auth{
		Password: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "output-auth"},
			Key:                  key,
		},
	}
}

// A credential that cannot be rendered on a single configuration line must fail the
// reconcile instead of being written into the generated configuration, where the part
// after the line break would become a directive of its own.
func TestResolveOutputAuthRejectsMultiLineCredentials(t *testing.T) {
	reconciler := newCredentialReconciler(t, map[string][]byte{
		"injected": []byte("attacker\n    uri /evil"),
		"trailing": []byte("s3cret\n"),
		"carriage": []byte("s3cret\r"),
		"valid":    []byte("s3cret"),
	})

	for _, key := range []string{"injected", "trailing", "carriage"} {
		t.Run(key, func(t *testing.T) {
			values := AuthValues{}
			err := reconciler.ResolveFluentbitOutputAuth("logging",
				OutputAuth{Values: &values, Enabled: true, Auth: passwordAuth(key)})
			if err == nil {
				t.Fatalf("a credential with a line break must be rejected, got %q", values.Password)
			}
			if !strings.Contains(err.Error(), "line break") {
				t.Errorf("the error must explain the line break rule, got: %v", err)
			}
			if strings.Contains(err.Error(), "s3cret") || strings.Contains(err.Error(), "attacker") {
				t.Errorf("the error must not disclose the credential, got: %v", err)
			}
		})
	}

	t.Run("fluentd applies the same rule", func(t *testing.T) {
		values := AuthValues{}
		err := reconciler.ResolveFluentdOutputAuth("logging",
			OutputAuth{Values: &values, Enabled: true, Auth: passwordAuth("injected")})
		if err == nil {
			t.Fatal("a credential with a line break must be rejected for FluentD too")
		}
	})

	t.Run("valid credential is resolved", func(t *testing.T) {
		values := AuthValues{}
		if err := reconciler.ResolveFluentbitOutputAuth("logging",
			OutputAuth{Values: &values, Enabled: true, Auth: passwordAuth("valid")}); err != nil {
			t.Fatal(err)
		}
		if values.Password != "s3cret" {
			t.Errorf("unexpected resolved password: %q", values.Password)
		}
	})
}

// Fluent Bit substitutes ${...} in a directive value with an environment variable, so
// a credential containing that sequence would reach the output with part of it
// replaced by an empty string.
func TestResolveFluentbitOutputAuthRejectsVariableReferences(t *testing.T) {
	reconciler := newCredentialReconciler(t, map[string][]byte{
		"expanded": []byte("pa${HOME}ss"),
		"dollar":   []byte("pa$$word"),
	})

	values := AuthValues{}
	err := reconciler.ResolveFluentbitOutputAuth("logging",
		OutputAuth{Values: &values, Enabled: true, Auth: passwordAuth("expanded")})
	if err == nil {
		t.Fatal(`a credential containing "${" must be rejected for Fluent Bit`)
	}

	t.Run("a plain dollar sign stays valid", func(t *testing.T) {
		values := AuthValues{}
		if err := reconciler.ResolveFluentbitOutputAuth("logging",
			OutputAuth{Values: &values, Enabled: true, Auth: passwordAuth("dollar")}); err != nil {
			t.Fatal(err)
		}
		if values.Password != "pa$$word" {
			t.Errorf("unexpected resolved password: %q", values.Password)
		}
	})

	t.Run("FluentD accepts the same value", func(t *testing.T) {
		values := AuthValues{}
		if err := reconciler.ResolveFluentdOutputAuth("logging",
			OutputAuth{Values: &values, Enabled: true, Auth: passwordAuth("expanded")}); err != nil {
			t.Fatalf("FluentD does not expand ${...}, so the value must be accepted: %v", err)
		}
	})
}

func TestResolveOutputAuthSkipsDisabledOutputs(t *testing.T) {
	reconciler := newCredentialReconciler(t, map[string][]byte{"valid": []byte("s3cret")})

	values := AuthValues{}
	if err := reconciler.ResolveFluentbitOutputAuth("logging",
		OutputAuth{Values: &values, Enabled: false, Auth: passwordAuth("missing-key")},
		OutputAuth{Values: &values, Enabled: true, Auth: nil}); err != nil {
		t.Fatal(err)
	}
	if values != (AuthValues{}) {
		t.Errorf("disabled outputs must not be resolved, got %#v", values)
	}
}
