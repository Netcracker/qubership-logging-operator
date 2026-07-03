package graylog

import (
	"context"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const adminSHA256 = "8c6976e5b5410415bde908bd4dee15dfb167a9c873fc4bb8a81f6f2ab448a918"

func TestCheckGraylog5(t *testing.T) {
	r := &GraylogReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Log: util.Logger("test-graylog"),
		},
	}

	tests := []struct {
		name     string
		image    string
		expected bool
	}{
		{"graylog 5.2.1", "graylog/graylog:5.2.1", true},
		{"graylog 4.3.0", "graylog/graylog:4.3.0", false},
		{"graylog 6.0.0", "graylog/graylog:6.0.0", false},
		{"registry with port", "registry:5000/graylog:5.0.0-rc1", true},
		{"latest tag no semver", "graylog/graylog:latest", false},
		{"no tag at all", "graylog", false},
		{"version 5.0.0.1 extra segments", "graylog:5.0.0.1", true},
		{"empty image", "", false},
		{"just version 5.0.0", "5.0.0", true},
		// Note: regex matches first semver in the entire string, not just the tag.
		// This is acceptable since real Docker images don't have semver in the path.
		{"version in path picks first semver", "5.0.0/graylog:4.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &loggingService.LoggingService{
				Spec: loggingService.LoggingServiceSpec{
					Graylog: &loggingService.Graylog{
						DockerImage: tt.image,
					},
				},
			}
			result := r.checkGraylog5(cr)
			if result != tt.expected {
				t.Errorf("checkGraylog5(%q) = %v, want %v", tt.image, result, tt.expected)
			}
		})
	}
}

func TestSetCredentialsLoadsSecretValues(t *testing.T) {
	r, cr := newGraylogTestReconciler(t, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "graylog-secret",
			Namespace: "logging",
		},
		Data: map[string][]byte{
			"user":              []byte("admin"),
			"password":          []byte("admin"),
			"elasticsearchHost": []byte("http://admin:admin@opensearch:9200"),
		},
	})

	if err := r.setCredentials(cr); err != nil {
		t.Fatalf("setCredentials() error = %v", err)
	}
	if cr.Spec.Graylog.User != "admin" {
		t.Fatalf("user = %q, want admin", cr.Spec.Graylog.User)
	}
	if cr.Spec.Graylog.Password != "admin" {
		t.Fatalf("password = %q, want admin", cr.Spec.Graylog.Password)
	}
	if cr.Spec.Graylog.ElasticsearchHost != "http://admin:admin@opensearch:9200" {
		t.Fatalf("elasticsearchHost = %q", cr.Spec.Graylog.ElasticsearchHost)
	}

	secret := &corev1.Secret{}
	if err := r.Client.Get(context.Background(), client.ObjectKey{Name: "graylog-secret", Namespace: "logging"}, secret); err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if got := string(secret.Data[util.GraylogSecretKeyRootPasswordSHA2]); got != adminSHA256 {
		t.Fatalf("rootPasswordSha2 = %q, want %q", got, adminSHA256)
	}
}

func TestSetCredentialsAllowsOpenSearchHostWithoutElasticsearchHost(t *testing.T) {
	r, cr := newGraylogTestReconciler(t, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "graylog-secret",
			Namespace: "logging",
		},
		Data: map[string][]byte{
			"user":     []byte("admin"),
			"password": []byte("admin"),
		},
	})
	cr.Spec.Graylog.OpenSearch = &loggingService.OpenSearch{
		Host: "http://admin:admin@opensearch:9200",
	}

	if err := r.setCredentials(cr); err != nil {
		t.Fatalf("setCredentials() error = %v", err)
	}
	if cr.Spec.Graylog.ElasticsearchHost != "" {
		t.Fatalf("elasticsearchHost = %q, want empty", cr.Spec.Graylog.ElasticsearchHost)
	}
}

func TestSetCredentialsRequiresElasticsearchHostWithoutOpenSearch(t *testing.T) {
	r, cr := newGraylogTestReconciler(t, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "graylog-secret",
			Namespace: "logging",
		},
		Data: map[string][]byte{
			"user":     []byte("admin"),
			"password": []byte("admin"),
		},
	})

	if err := r.setCredentials(cr); err == nil {
		t.Fatal("setCredentials() error = nil, want error")
	}
}

func TestSetCredentialsRequiresUser(t *testing.T) {
	r, cr := newGraylogTestReconciler(t, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "graylog-secret",
			Namespace: "logging",
		},
		Data: map[string][]byte{
			"password":          []byte("admin"),
			"elasticsearchHost": []byte("http://admin:admin@opensearch:9200"),
		},
	})

	if err := r.setCredentials(cr); err == nil {
		t.Fatal("setCredentials() error = nil, want error")
	}
}

func TestSetCredentialsRequiresPassword(t *testing.T) {
	r, cr := newGraylogTestReconciler(t, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "graylog-secret",
			Namespace: "logging",
		},
		Data: map[string][]byte{
			"user":              []byte("admin"),
			"elasticsearchHost": []byte("http://admin:admin@opensearch:9200"),
		},
	})

	if err := r.setCredentials(cr); err == nil {
		t.Fatal("setCredentials() error = nil, want error")
	}
}

func newGraylogTestReconciler(t *testing.T, objects ...client.Object) (*GraylogReconciler, *loggingService.LoggingService) {
	t.Helper()

	testScheme := runtime.NewScheme()
	if err := loggingService.AddToScheme(testScheme); err != nil {
		t.Fatalf("add logging service scheme: %v", err)
	}
	if err := corev1.AddToScheme(testScheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}

	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "logging-service",
			Namespace: "logging",
		},
		Spec: loggingService.LoggingServiceSpec{
			Graylog: &loggingService.Graylog{
				GraylogSecretName: "graylog-secret",
			},
		},
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(objects...).
		Build()
	r := &GraylogReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: fakeClient,
			Scheme: testScheme,
			Log:    util.Logger("test-graylog"),
		},
	}
	return r, cr
}
