package reconcile

import (
	"context"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatalf("add corev1: %v", err)
	}
	if err := loggingService.AddToScheme(s); err != nil {
		t.Fatalf("add loggingService: %v", err)
	}
	return s
}

func newCR() *loggingService.LoggingService {
	return &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "ls", Namespace: "logging", UID: "uid-1"},
	}
}

func TestServiceCreateThenUpdatePreservesClusterIP(t *testing.T) {
	scheme := newScheme(t)
	cr := newCR()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).Build()

	desired := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "svc", Namespace: "logging"},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{Name: "http", Port: 80}},
		},
	}
	if err := Service(context.Background(), c, scheme, cr, desired); err != nil {
		t.Fatalf("Service create: %v", err)
	}

	// Simulate API server assigning ClusterIP.
	got := &corev1.Service{}
	if err := c.Get(context.Background(), client_objKey("logging", "svc"), got); err != nil {
		t.Fatalf("get after create: %v", err)
	}
	got.Spec.ClusterIP = "10.96.0.42"
	if err := c.Update(context.Background(), got); err != nil {
		t.Fatalf("seed clusterIP: %v", err)
	}

	// Re-apply with empty ClusterIP — Service() should preserve the assigned one.
	desired2 := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "svc", Namespace: "logging"},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{Name: "http", Port: 8080}},
		},
	}
	if err := Service(context.Background(), c, scheme, cr, desired2); err != nil {
		t.Fatalf("Service update: %v", err)
	}

	final := &corev1.Service{}
	if err := c.Get(context.Background(), client_objKey("logging", "svc"), final); err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if final.Spec.ClusterIP != "10.96.0.42" {
		t.Fatalf("ClusterIP not preserved: got %q", final.Spec.ClusterIP)
	}
	if len(final.Spec.Ports) != 1 || final.Spec.Ports[0].Port != 8080 {
		t.Fatalf("port update not applied: got %+v", final.Spec.Ports)
	}
}

func TestServiceAccountCreateIsIdempotent(t *testing.T) {
	scheme := newScheme(t)
	cr := newCR()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).Build()

	sa := &corev1.ServiceAccount{
		TypeMeta:   metav1.TypeMeta{Kind: "ServiceAccount", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "sa", Namespace: "logging"},
	}
	if err := ServiceAccount(context.Background(), c, scheme, cr, sa); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	sa2 := sa.DeepCopy()
	sa2.ResourceVersion = ""
	if err := ServiceAccount(context.Background(), c, scheme, cr, sa2); err != nil {
		t.Fatalf("second apply: %v", err)
	}
}
