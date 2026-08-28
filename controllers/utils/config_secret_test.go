package utils

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEqualConfigSecret(t *testing.T) {
	tests := []struct {
		name     string
		source   *corev1.Secret
		target   *corev1.Secret
		expected bool
	}{
		{
			name: "same data and labels",
			source: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "fluent-bit"}},
				Data:       map[string][]byte{"key": []byte("value")},
			},
			target: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "fluent-bit"}},
				Data:       map[string][]byte{"key": []byte("value")},
			},
			expected: true,
		},
		{
			name:     "different data",
			source:   &corev1.Secret{Data: map[string][]byte{"key": []byte("value1")}},
			target:   &corev1.Secret{Data: map[string][]byte{"key": []byte("value2")}},
			expected: false,
		},
		{
			name: "different labels",
			source: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"env": "prod"}},
				Data:       map[string][]byte{"key": []byte("value")},
			},
			target: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"env": "dev"}},
				Data:       map[string][]byte{"key": []byte("value")},
			},
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := EqualConfigSecret(test.source, test.target); got != test.expected {
				t.Errorf("EqualConfigSecret() = %v, want %v", got, test.expected)
			}
		})
	}
}

func TestDeleteLegacyConfigMap(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	legacy := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-fluentbit", Namespace: "logging"},
		Data:       map[string]string{"fluent-bit.conf": "old"},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(legacy).Build()
	reconciler := &ComponentReconciler{Client: fakeClient, Scheme: scheme, Log: Logger("test-config-secret")}

	if err := reconciler.DeleteLegacyConfigMap("logging", "logging-fluentbit"); err != nil {
		t.Fatal(err)
	}
	err := fakeClient.Get(t.Context(), client.ObjectKeyFromObject(legacy), &corev1.ConfigMap{})
	if !errors.IsNotFound(err) {
		t.Fatalf("the legacy ConfigMap must be deleted, got error %v", err)
	}

	// Deleting an already absent ConfigMap is a no-op.
	if err := reconciler.DeleteLegacyConfigMap("logging", "logging-fluentbit"); err != nil {
		t.Fatalf("deleting an absent ConfigMap must not fail: %v", err)
	}
}
