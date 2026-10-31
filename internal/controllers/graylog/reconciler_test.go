package graylog

import (
	"context"
	"reflect"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const adminSHA256 = "8c6976e5b5410415bde908bd4dee15dfb167a9c873fc4bb8a81f6f2ab448a918"

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

func TestDeleteUpgradeJobsRemovesLeftoverMongoDBUpgradeJobs(t *testing.T) {
	job := func(name string) *batchv1.Job {
		return &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "logging"}}
	}
	// An earlier upgrade may have left only some of the Jobs.
	r, cr := newGraylogTestReconciler(t,
		job("mongo-upgrade-job-40"),
		job("mongo-upgrade-job-50"),
		job("unrelated-job"),
	)

	if err := r.deleteUpgradeJobs(cr); err != nil {
		t.Fatalf("delete upgrade Jobs: %v", err)
	}

	jobs := &batchv1.JobList{}
	if err := r.Client.List(context.Background(), jobs, client.InNamespace("logging")); err != nil {
		t.Fatalf("list Jobs: %v", err)
	}
	var names []string
	for _, j := range jobs.Items {
		names = append(names, j.Name)
	}
	if !reflect.DeepEqual(names, []string{"unrelated-job"}) {
		t.Fatalf("Jobs after cleanup = %v, want only unrelated-job", names)
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
	if err := batchv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("add batch scheme: %v", err)
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
