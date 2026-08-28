package fluentbit

import (
	"context"
	"reflect"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHandleDaemonSetUpdatesTheCompletePodSpec(t *testing.T) {
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			OpenshiftDeploy: true,
			Fluentbit: &loggingService.Fluentbit{
				DockerImage:       "fluent-bit:test",
				PriorityClassName: "system-cluster-critical",
				ConfigmapReload:   &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
			},
		},
	}
	dynamicParameters := util.DynamicParameters{ContainerRuntimeType: "containerd"}
	desired, err := fluentbitDaemonSet(cr, dynamicParameters)
	if err != nil {
		t.Fatalf("render Fluent Bit DaemonSet: %v", err)
	}
	existing := desired.DeepCopy()
	existing.Labels = nil
	existing.Spec.Template.Spec = corev1.PodSpec{PriorityClassName: "stale-priority"}

	testScheme := runtime.NewScheme()
	if err := loggingService.AddToScheme(testScheme); err != nil {
		t.Fatalf("add LoggingService scheme: %v", err)
	}
	if err := appsv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("add apps scheme: %v", err)
	}
	testClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(existing).Build()
	reconciler := &FluentbitReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: testClient,
			Scheme: testScheme,
			Log:    util.Logger("test-fluentbit-update"),
		},
		DynamicParameters: dynamicParameters,
	}

	if err := reconciler.handleDaemonSet(cr); err != nil {
		t.Fatalf("update Fluent Bit DaemonSet: %v", err)
	}
	updated := &appsv1.DaemonSet{}
	key := types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}
	if err := testClient.Get(context.Background(), key, updated); err != nil {
		t.Fatalf("get updated Fluent Bit DaemonSet: %v", err)
	}
	if !reflect.DeepEqual(updated.Spec.Template.Spec, desired.Spec.Template.Spec) {
		t.Error("updated pod spec does not match the rendered pod spec")
	}
}

func TestHandleDaemonSetCreatesMissingDaemonSet(t *testing.T) {
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{Fluentbit: &loggingService.Fluentbit{
			DockerImage:     "fluent-bit:test",
			ConfigmapReload: &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
		}},
	}
	testScheme := runtime.NewScheme()
	if err := loggingService.AddToScheme(testScheme); err != nil {
		t.Fatalf("add LoggingService scheme: %v", err)
	}
	if err := appsv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("add apps scheme: %v", err)
	}
	testClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	reconciler := &FluentbitReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: testClient,
			Scheme: testScheme,
			Log:    util.Logger("test-fluentbit-create"),
		},
		DynamicParameters: util.DynamicParameters{ContainerRuntimeType: "containerd"},
	}

	if err := reconciler.handleDaemonSet(cr); err != nil {
		t.Fatalf("create Fluent Bit DaemonSet: %v", err)
	}
	created := &appsv1.DaemonSet{}
	key := types.NamespacedName{Name: util.FluentbitComponentName, Namespace: cr.Namespace}
	if err := testClient.Get(context.Background(), key, created); err != nil {
		t.Fatalf("get created Fluent Bit DaemonSet: %v", err)
	}
}

func TestHandleDaemonSetReturnsManifestError(t *testing.T) {
	reconciler := newTestFluentbitReconciler()
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging"},
	}

	if err := reconciler.handleDaemonSet(cr); err == nil {
		t.Fatal("expected an error when Fluent Bit configuration is missing")
	}
}

func TestUpdateDaemonSetReturnsGetError(t *testing.T) {
	testScheme := runtime.NewScheme()
	if err := appsv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("add apps scheme: %v", err)
	}
	reconciler := &FluentbitReconciler{ComponentReconciler: &util.ComponentReconciler{
		Client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
		Scheme: testScheme,
		Log:    util.Logger("test-fluentbit-get-error"),
	}}
	desired := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "missing", Namespace: "logging"}}

	if err := reconciler.updateDaemonSet(desired); err == nil {
		t.Fatal("expected an error when the existing DaemonSet is missing")
	}
}

func newTestFluentbitReconciler() *FluentbitReconciler {
	return &FluentbitReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Log: util.Logger("test-fluentbit"),
		},
	}
}

func TestFluentbitEqual(t *testing.T) {
	r := newTestFluentbitReconciler()

	t.Run("same data returns true", func(t *testing.T) {
		a := &corev1.ConfigMap{Data: map[string]string{"key": "value"}}
		b := &corev1.ConfigMap{Data: map[string]string{"key": "value"}}
		if !r.Equal(a, b) {
			t.Error("expected equal for same data")
		}
	})

	t.Run("different data returns false", func(t *testing.T) {
		a := &corev1.ConfigMap{Data: map[string]string{"key": "value1"}}
		b := &corev1.ConfigMap{Data: map[string]string{"key": "value2"}}
		if r.Equal(a, b) {
			t.Error("expected not equal for different data")
		}
	})

	t.Run("different binary data returns false", func(t *testing.T) {
		a := &corev1.ConfigMap{BinaryData: map[string][]byte{"key": {1, 2}}}
		b := &corev1.ConfigMap{BinaryData: map[string][]byte{"key": {3, 4}}}
		if r.Equal(a, b) {
			t.Error("expected not equal for different binary data")
		}
	})

	t.Run("different labels still returns true (fluentbit ignores labels)", func(t *testing.T) {
		a := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"env": "prod"}},
			Data:       map[string]string{"key": "value"},
		}
		b := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"env": "dev"}},
			Data:       map[string]string{"key": "value"},
		}
		if !r.Equal(a, b) {
			t.Error("fluentbit Equal should ignore labels, but it didn't")
		}
	})
}
