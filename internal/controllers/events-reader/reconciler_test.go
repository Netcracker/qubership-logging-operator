package events_reader

import (
	"context"
	"reflect"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder/eventsreader"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHandleDeploymentUpdatesVolumesAndPriorityClass(t *testing.T) {
	cr := newEventsReaderLoggingService(false)
	cr.Spec.CloudEventsReader.PriorityClassName = "system-cluster-critical"
	desired := eventsreader.Deployment(cr, defaults.Get())
	existing := desired.DeepCopy()
	existing.Spec.Template.Spec.Volumes = nil
	existing.Spec.Template.Spec.PriorityClassName = "stale-priority"

	testScheme := runtime.NewScheme()
	if err := loggingService.AddToScheme(testScheme); err != nil {
		t.Fatalf("add LoggingService scheme: %v", err)
	}
	if err := appsv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("add apps scheme: %v", err)
	}
	testClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(existing).Build()
	reconciler := &EventsReaderReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: testClient,
			Scheme: testScheme,
			Log:    util.Logger("test-events-reader-update"),
		},
		Defaults: defaults.Get(),
	}

	if err := reconciler.handleDeployment(context.Background(), cr); err != nil {
		t.Fatalf("update Events Reader Deployment: %v", err)
	}
	updated := &appsv1.Deployment{}
	key := types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}
	if err := testClient.Get(context.Background(), key, updated); err != nil {
		t.Fatalf("get updated Events Reader Deployment: %v", err)
	}
	if !reflect.DeepEqual(updated.Spec.Template.Spec.Volumes, desired.Spec.Template.Spec.Volumes) {
		t.Error("updated volumes do not match the rendered volumes")
	}
	if updated.Spec.Template.Spec.PriorityClassName != desired.Spec.Template.Spec.PriorityClassName {
		t.Errorf("updated priority class = %q, want %q", updated.Spec.Template.Spec.PriorityClassName,
			desired.Spec.Template.Spec.PriorityClassName)
	}
}

func newEventsReaderLoggingService(openshift bool) *loggingService.LoggingService {
	return &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			CloudURL:        "https://kubernetes.default.svc",
			OpenshiftDeploy: openshift,
			CloudEventsReader: &loggingService.CloudEventsReader{
				DockerImage: "ghcr.io/netcracker/qubership-kube-events-reader:main",
			},
		},
	}
}
