package fluentbit_forwarder_aggregator

import (
	"context"
	"reflect"
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHandleForwarderDaemonSetUpdatesTheCompletePodSpec(t *testing.T) {
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
	desired, err := forwarderDaemonSet(cr, dynamicParameters)
	if err != nil {
		t.Fatalf("render Fluent Bit forwarder DaemonSet: %v", err)
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
	reconciler := &HAFluentReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: testClient,
			Scheme: testScheme,
			Log:    util.Logger("test-ha-fluent-forwarder-update"),
		},
		DynamicParameters: dynamicParameters,
	}

	if err := reconciler.handleForwarderDaemonSet(cr); err != nil {
		t.Fatalf("update Fluent Bit forwarder DaemonSet: %v", err)
	}
	updated := &appsv1.DaemonSet{}
	key := types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}
	if err := testClient.Get(context.Background(), key, updated); err != nil {
		t.Fatalf("get updated Fluent Bit forwarder DaemonSet: %v", err)
	}
	if !reflect.DeepEqual(updated.Spec.Template.Spec, desired.Spec.Template.Spec) {
		t.Error("updated pod spec does not match the rendered pod spec")
	}
}

func TestHandleForwarderDaemonSetCreatesMissingDaemonSet(t *testing.T) {
	cr := newHAFluentHandlerLoggingService()
	testScheme := newHAFluentHandlerTestScheme(t)
	testClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	reconciler := &HAFluentReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: testClient,
			Scheme: testScheme,
			Log:    util.Logger("test-ha-fluent-forwarder-create"),
		},
		DynamicParameters: util.DynamicParameters{ContainerRuntimeType: "containerd"},
	}

	if err := reconciler.handleForwarderDaemonSet(cr); err != nil {
		t.Fatalf("create Fluent Bit forwarder DaemonSet: %v", err)
	}
	created := &appsv1.DaemonSet{}
	key := types.NamespacedName{Name: util.ForwarderFluentbitComponentName, Namespace: cr.Namespace}
	if err := testClient.Get(context.Background(), key, created); err != nil {
		t.Fatalf("get created Fluent Bit forwarder DaemonSet: %v", err)
	}
}

func TestHandleForwarderDaemonSetReturnsManifestError(t *testing.T) {
	reconciler := newTestHAFluentReconciler()
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging"},
	}

	if err := reconciler.handleForwarderDaemonSet(cr); err == nil {
		t.Fatal("expected an error when Fluent Bit configuration is missing")
	}
}

func TestUpdateForwarderDaemonSetReturnsGetError(t *testing.T) {
	testScheme := runtime.NewScheme()
	if err := appsv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("add apps scheme: %v", err)
	}
	reconciler := &HAFluentReconciler{ComponentReconciler: &util.ComponentReconciler{
		Client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
		Scheme: testScheme,
		Log:    util.Logger("test-ha-fluent-forwarder-get-error"),
	}}
	desired := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "missing", Namespace: "logging"}}

	if err := reconciler.updateForwarderDaemonSet(desired); err == nil {
		t.Fatal("expected an error when the existing DaemonSet is missing")
	}
}

func TestCreateOrUpdateAggregatorStatefulSetCreatesMissingStatefulSet(t *testing.T) {
	cr := newHAFluentHandlerLoggingService()
	desired, err := aggregatorStatefulSet(cr)
	if err != nil {
		t.Fatalf("render Fluent Bit aggregator StatefulSet: %v", err)
	}
	testScheme := newHAFluentHandlerTestScheme(t)
	testClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	reconciler := &HAFluentReconciler{ComponentReconciler: &util.ComponentReconciler{
		Client: testClient,
		Scheme: testScheme,
		Log:    util.Logger("test-ha-fluent-aggregator-create"),
	}}

	if err := reconciler.createOrUpdateAggregatorStatefulSet(cr, desired); err != nil {
		t.Fatalf("create Fluent Bit aggregator StatefulSet: %v", err)
	}
}

func TestHandleAggregatorStatefulSetUpdatesAndWaitsForReadiness(t *testing.T) {
	cr := newHAFluentHandlerLoggingService()
	desired, err := aggregatorStatefulSet(cr)
	if err != nil {
		t.Fatalf("render Fluent Bit aggregator StatefulSet: %v", err)
	}
	existing := desired.DeepCopy()
	existing.Labels = nil
	existing.Spec.Template.Spec.Containers = nil
	existing.Status.Replicas = *desired.Spec.Replicas
	existing.Status.ReadyReplicas = *desired.Spec.Replicas
	testScheme := newHAFluentHandlerTestScheme(t)
	testClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(existing).Build()
	reconciler := &HAFluentReconciler{ComponentReconciler: &util.ComponentReconciler{
		Client: testClient,
		Scheme: testScheme,
		Log:    util.Logger("test-ha-fluent-aggregator-update"),
	}}

	initialDelay := util.InitialDelay
	util.InitialDelay = 0
	defer func() { util.InitialDelay = initialDelay }()

	if err := reconciler.handleAggregatorStatefulSet(cr); err != nil {
		t.Fatalf("update Fluent Bit aggregator StatefulSet: %v", err)
	}
	updated := &appsv1.StatefulSet{}
	key := types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}
	if err := testClient.Get(context.Background(), key, updated); err != nil {
		t.Fatalf("get updated Fluent Bit aggregator StatefulSet: %v", err)
	}
	if !reflect.DeepEqual(updated.Spec.Template.Spec.Containers, desired.Spec.Template.Spec.Containers) {
		t.Error("updated containers do not match the rendered containers")
	}
}

func TestHandleAggregatorStatefulSetReturnsCreateError(t *testing.T) {
	cr := newHAFluentHandlerLoggingService()
	reconciler := &HAFluentReconciler{ComponentReconciler: &util.ComponentReconciler{
		Client: fake.NewClientBuilder().Build(),
		Scheme: runtime.NewScheme(),
		Log:    util.Logger("test-ha-fluent-aggregator-create-error"),
	}}

	if err := reconciler.handleAggregatorStatefulSet(cr); err == nil {
		t.Fatal("expected an error when the owner type is not registered")
	}
}

func TestWaitForAggregatorReturnsClientError(t *testing.T) {
	cr := newHAFluentHandlerLoggingService()
	testScheme := newHAFluentHandlerTestScheme(t)
	testClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	reconciler := &HAFluentReconciler{ComponentReconciler: &util.ComponentReconciler{
		Client: testClient,
		Scheme: testScheme,
		Log:    util.Logger("test-ha-fluent-aggregator-wait-error"),
	}}

	initialDelay := util.InitialDelay
	util.InitialDelay = 0
	defer func() { util.InitialDelay = initialDelay }()

	if err := reconciler.waitForAggregator(cr); err == nil {
		t.Fatal("expected an error when the aggregator StatefulSet is missing")
	}
}

func newHAFluentHandlerLoggingService() *loggingService.LoggingService {
	resources := &corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
	}
	return &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{Fluentbit: &loggingService.Fluentbit{
			DockerImage:     "fluent-bit:test",
			ConfigmapReload: &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
			Aggregator: &loggingService.FluentbitAggregator{
				DockerImage:     "fluent-bit:test",
				StartupTimeout:  1,
				Resources:       resources,
				ConfigmapReload: &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
			},
		}},
	}
}

func newHAFluentHandlerTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	testScheme := runtime.NewScheme()
	if err := loggingService.AddToScheme(testScheme); err != nil {
		t.Fatalf("add LoggingService scheme: %v", err)
	}
	if err := appsv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("add apps scheme: %v", err)
	}
	return testScheme
}

func newTestHAFluentReconciler() *HAFluentReconciler {
	return &HAFluentReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Log: util.Logger("test-ha-fluent"),
		},
	}
}

func TestHAFluentEqual(t *testing.T) {
	r := newTestHAFluentReconciler()

	t.Run("same data and labels returns true", func(t *testing.T) {
		a := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "fluent"}},
			Data:       map[string]string{"key": "value"},
		}
		b := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "fluent"}},
			Data:       map[string]string{"key": "value"},
		}
		if !r.Equal(a, b) {
			t.Error("expected equal for same data and labels")
		}
	})

	t.Run("different data returns false", func(t *testing.T) {
		a := &corev1.ConfigMap{Data: map[string]string{"key": "value1"}}
		b := &corev1.ConfigMap{Data: map[string]string{"key": "value2"}}
		if r.Equal(a, b) {
			t.Error("expected not equal for different data")
		}
	})

	t.Run("different labels returns false (HA-fluent checks labels)", func(t *testing.T) {
		a := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"env": "prod"}},
			Data:       map[string]string{"key": "value"},
		}
		b := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"env": "dev"}},
			Data:       map[string]string{"key": "value"},
		}
		if r.Equal(a, b) {
			t.Error("HA-fluent Equal should detect label changes, but it didn't")
		}
	})
}

// Verifies that resolveAggregatorOutputCredentials correctly resolves Auth references
// into actual values from a Kubernetes Secret, and that these values are inlined into
// the rendered aggregator config Secret (output-http.conf) instead of being left unset.
func TestResolveAggregatorOutputCredentials(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "output-auth", Namespace: "logging"},
		Data: map[string][]byte{
			"username": []byte("aggregator-user"),
			"password": []byte("aggregator-password"),
			"token":    []byte("aggregator-token"),
		},
	}
	reconciler := &HAFluentReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build(),
			Log:    util.Logger("test-ha-fluent"),
		},
	}
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				Aggregator: &loggingService.FluentbitAggregator{
					Output: &loggingService.OutputFluentbit{
						Http: &loggingService.HttpFluentbit{
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
		},
	}

	credentials, err := reconciler.resolveAggregatorOutputCredentials(cr)
	if err != nil {
		t.Fatal(err)
	}
	if credentials.Http.Token != "aggregator-token" ||
		credentials.Http.User != "aggregator-user" ||
		credentials.Http.Password != "aggregator-password" {
		t.Fatalf("unexpected resolved credentials: %#v", credentials.Http)
	}

	configSecret, err := aggregatorConfigSecret(cr, util.DynamicParameters{}, credentials)
	if err != nil {
		t.Fatal(err)
	}
	httpOutput := string(configSecret.Data["output-http.conf"])
	for _, expected := range []string{"aggregator-user", "aggregator-password", "Bearer aggregator-token"} {
		if !strings.Contains(httpOutput, expected) {
			t.Errorf("generated HTTP output does not contain %q", expected)
		}
	}
}

// Regression test for the fix that sets ResourceVersion on the desired Secret before
// updating: without it, UpdateResource fails against a real/fake client because the
// object it's given has no ResourceVersion set.
func TestUpdateSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := loggingService.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-fluentbit-aggregator", Namespace: "logging"},
		Data:       map[string][]byte{"fluent-bit.conf": []byte("old")},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	reconciler := &HAFluentReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: fakeClient,
			Scheme: scheme,
			Log:    util.Logger("test-ha-fluent"),
		},
	}
	cr := &loggingService.LoggingService{
		TypeMeta:   metav1.TypeMeta{APIVersion: loggingService.GroupVersion.String(), Kind: "LoggingService"},
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging", UID: "test-uid"},
	}
	desired := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-fluentbit-aggregator", Namespace: "logging"},
		Data:       map[string][]byte{"fluent-bit.conf": []byte("new")},
	}

	updated, err := reconciler.updateSecret(cr, desired)
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
	if string(actual.Data["fluent-bit.conf"]) != "new" {
		t.Fatalf("unexpected Secret data: %q", actual.Data["fluent-bit.conf"])
	}
}
