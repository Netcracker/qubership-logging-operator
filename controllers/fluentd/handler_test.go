package fluentd

import (
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestFluentdReconciler() *FluentdReconciler {
	return &FluentdReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Log: util.Logger("test-fluentd"),
		},
	}
}

func TestResolveOutputCredentials(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "output-auth", Namespace: "logging"},
		Data: map[string][]byte{
			"username": []byte("fluentd-user"),
			"password": []byte("fluentd-password"),
			"token":    []byte("fluentd-token"),
		},
	}
	reconciler := &FluentdReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build(),
			Log:    util.Logger("test-fluentd"),
		},
	}
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentd: &loggingService.Fluentd{
				Output: &loggingService.OutputFluentd{
					Http: &loggingService.HttpFluentd{
						Enabled: true,
						Auth: &loggingService.Auth{
							Token:    &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "output-auth"}, Key: "token"},
							User:     &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "output-auth"}, Key: "username"},
							Password: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "output-auth"}, Key: "password"},
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
	if credentials.Http.Token != "fluentd-token" ||
		credentials.Http.User != "fluentd-user" ||
		credentials.Http.Password != "fluentd-password" {
		t.Fatalf("unexpected resolved credentials: %#v", credentials.Http)
	}

	configSecret, err := fluentdConfigSecret(cr, util.DynamicParameters{}, credentials)
	if err != nil {
		t.Fatal(err)
	}
	httpOutput := string(configSecret.Data["output-http.conf"])
	for _, expected := range []string{"fluentd-user", "fluentd-password", "Bearer fluentd-token"} {
		if !strings.Contains(httpOutput, expected) {
			t.Errorf("generated HTTP output does not contain %q", expected)
		}
	}
}

func TestCreateOrUpdateConfigSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := loggingService.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-fluentd", Namespace: "logging"},
		Data:       map[string][]byte{"fluent.conf": []byte("old")},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	reconciler := &FluentdReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: fakeClient,
			Scheme: scheme,
			Log:    util.Logger("test-fluentd"),
		},
	}
	cr := &loggingService.LoggingService{
		TypeMeta:   metav1.TypeMeta{APIVersion: loggingService.GroupVersion.String(), Kind: "LoggingService"},
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging", UID: "test-uid"},
	}
	desired := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-fluentd", Namespace: "logging"},
		Data:       map[string][]byte{"fluent.conf": []byte("new")},
	}

	updated, err := reconciler.CreateOrUpdateConfigSecret(cr, desired)
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("expected the configuration Secret to be updated")
	}
	actual := &corev1.Secret{}
	if err := fakeClient.Get(t.Context(), client.ObjectKeyFromObject(desired), actual); err != nil {
		t.Fatal(err)
	}
	if string(actual.Data["fluent.conf"]) != "new" {
		t.Fatalf("unexpected Secret data: %q", actual.Data["fluent.conf"])
	}
}

func TestFluentdConfigSecretHTTPHeadersWithToken(t *testing.T) {
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentd: &loggingService.Fluentd{
				Output: &loggingService.OutputFluentd{
					Http: &loggingService.HttpFluentd{
						Enabled: true,
						Headers: map[string]string{"X-Scope-OrgID": "tenant-1"},
						Auth: &loggingService.Auth{
							Token: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "output-auth"},
								Key:                  "token",
							},
						},
					},
				},
			},
		},
	}
	credentials := outputCredentials{Http: util.AuthValues{Token: "fluentd-token"}}

	secret, err := fluentdConfigSecret(cr, util.DynamicParameters{}, credentials)
	if err != nil {
		t.Fatalf("custom headers combined with a bearer token must render: %v", err)
	}
	httpOutput := string(secret.Data["output-http.conf"])
	for _, expected := range []string{`"X-Scope-OrgID":"tenant-1"`, `"Authorization":"Bearer fluentd-token"`} {
		if !strings.Contains(httpOutput, expected) {
			t.Errorf("generated HTTP output does not contain %q", expected)
		}
	}
}

func TestFluentdConfigSecretCredentialsAreNotInterpolated(t *testing.T) {
	selector := func(key string) *corev1.SecretKeySelector {
		return &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "output-auth"},
			Key:                  key,
		}
	}
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentd: &loggingService.Fluentd{
				Output: &loggingService.OutputFluentd{
					Loki: &loggingService.LokiFluentd{
						Enabled: true,
						Auth: &loggingService.Auth{
							User:     selector("username"),
							Password: selector("password"),
						},
					},
					Http: &loggingService.HttpFluentd{
						Enabled: true,
						Auth: &loggingService.Auth{
							User:     selector("username"),
							Password: selector("password"),
						},
					},
				},
			},
		},
	}
	// A password that FluentD would evaluate as embedded Ruby inside a double-quoted string.
	password := `p#{exec("id")}a`
	credentials := outputCredentials{
		Loki: util.AuthValues{User: "loki-user", Password: password},
		Http: util.AuthValues{User: "http-user", Password: password},
	}

	secret, err := fluentdConfigSecret(cr, util.DynamicParameters{}, credentials)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"output-loki.conf", "output-http.conf"} {
		rendered := string(secret.Data[file])
		if !strings.Contains(rendered, `password   'p#{exec("id")}a'`) &&
			!strings.Contains(rendered, `password 'p#{exec("id")}a'`) {
			t.Errorf("%s must quote the password with single quotes, got:\n%s", file, rendered)
		}
		if strings.Contains(rendered, `password "`) || strings.Contains(rendered, `password   "`) {
			t.Errorf("%s must not render the password as a double-quoted string, got:\n%s", file, rendered)
		}
	}
}
