package fluentbit

import (
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestFluentbitReconciler() *FluentbitReconciler {
	return &FluentbitReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Log: util.Logger("test-fluentbit"),
		},
	}
}

func TestFluentbitEqual(t *testing.T) {
	r := newTestFluentbitReconciler()

	t.Run("same data and labels returns true", func(t *testing.T) {
		a := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "fluentbit"}},
			Data:       map[string][]byte{"key": []byte("value")},
		}
		b := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "fluentbit"}},
			Data:       map[string][]byte{"key": []byte("value")},
		}
		if !r.Equal(a, b) {
			t.Error("expected equal for same data and labels")
		}
	})

	t.Run("different data returns false", func(t *testing.T) {
		a := &corev1.Secret{Data: map[string][]byte{"key": []byte("value1")}}
		b := &corev1.Secret{Data: map[string][]byte{"key": []byte("value2")}}
		if r.Equal(a, b) {
			t.Error("expected not equal for different data")
		}
	})

	t.Run("different labels returns false", func(t *testing.T) {
		a := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"env": "prod"}},
			Data:       map[string][]byte{"key": []byte("value")},
		}
		b := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"env": "dev"}},
			Data:       map[string][]byte{"key": []byte("value")},
		}
		if r.Equal(a, b) {
			t.Error("fluentbit Equal should detect label changes, but it didn't")
		}
	})
}

// Verifies that resolveOutputCredentials correctly resolves Auth references
// (SecretKeySelector for username/password/token) into actual values from a Kubernetes
// Secret, and that these values are inlined into the rendered config Secret
// (output-http.conf, output-loki.conf) instead of ${LOKI_USERNAME}-style placeholders.
func TestResolveOutputCredentials(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "output-auth", Namespace: "logging"},
		Data: map[string][]byte{
			"username": []byte("fluentbit-user"),
			"password": []byte("fluentbit-password"),
			"token":    []byte("fluentbit-token"),
		},
	}
	reconciler := &FluentbitReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build(),
			Log:    util.Logger("test-fluentbit"),
		},
	}
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				Output: &loggingService.OutputFluentbit{
					Http: &loggingService.HttpFluentbit{
						Enabled: true,
						Auth: &loggingService.Auth{
							Token:    &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "output-auth"}, Key: "token"},
							User:     &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "output-auth"}, Key: "username"},
							Password: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "output-auth"}, Key: "password"},
						},
					},
					Loki: &loggingService.LokiFluentbit{
						Enabled: true,
						Auth: &loggingService.Auth{
							Token: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "output-auth"}, Key: "token"},
						},
					},
				},
			},
		},
	}

	credentials, err := reconciler.resolveOutputCredentials(cr)
	if err != nil {
		t.Fatal(err)
	}
	if credentials.Http.Token != "fluentbit-token" ||
		credentials.Http.User != "fluentbit-user" ||
		credentials.Http.Password != "fluentbit-password" {
		t.Fatalf("unexpected resolved HTTP credentials: %#v", credentials.Http)
	}
	if credentials.Loki.Token != "fluentbit-token" {
		t.Fatalf("unexpected resolved Loki credentials: %#v", credentials.Loki)
	}

	configSecret, err := fluentbitConfigSecret(cr, util.DynamicParameters{}, credentials)
	if err != nil {
		t.Fatal(err)
	}
	httpOutput := string(configSecret.Data["output-http.conf"])
	for _, expected := range []string{"fluentbit-user", "fluentbit-password", "Bearer fluentbit-token"} {
		if !strings.Contains(httpOutput, expected) {
			t.Errorf("generated HTTP output does not contain %q", expected)
		}
	}
	lokiOutput := string(configSecret.Data["output-loki.conf"])
	if !strings.Contains(lokiOutput, "fluentbit-token") {
		t.Errorf("generated Loki output does not contain resolved token")
	}
}
