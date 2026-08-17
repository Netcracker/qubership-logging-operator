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
	api_errors "k8s.io/apimachinery/pkg/api/errors"
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

func TestForwarderDaemonSetHardeningExceptions(t *testing.T) {
	cr := &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				DockerImage:     "fluent-bit:test",
				ConfigmapReload: &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
			},
		},
	}

	daemonSet, err := forwarderDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
	if err != nil {
		t.Fatalf("render Fluent Bit forwarder DaemonSet: %v", err)
	}

	podSpec := daemonSet.Spec.Template.Spec
	assertForwarderPodContext(t, podSpec.SecurityContext)
	assertForwarderContainerSecurity(t, podSpec.Containers[1])

	cr.Spec.OpenshiftDeploy = true
	openShiftDaemonSet, err := forwarderDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
	if err != nil {
		t.Fatalf("render OpenShift Fluent Bit forwarder DaemonSet: %v", err)
	}
	assertOpenShiftForwarderSecurity(t, openShiftDaemonSet.Spec.Template.Spec)
}

func assertForwarderPodContext(t *testing.T, context *corev1.PodSecurityContext) {
	t.Helper()
	if context == nil || context.SeccompProfile == nil ||
		context.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("pod must use the RuntimeDefault seccomp profile")
	}
}

func assertForwarderContainerSecurity(t *testing.T, forwarder corev1.Container) {
	t.Helper()
	context := forwarder.SecurityContext
	if context == nil {
		t.Fatal("forwarder security context is missing")
	}
	if context.RunAsUser == nil || *context.RunAsUser != 0 || context.RunAsNonRoot == nil || *context.RunAsNonRoot ||
		context.RunAsGroup == nil || *context.RunAsGroup != 0 {
		t.Error("forwarder must run as root to access node logs and its existing state under /var/log")
	}
	if context.ReadOnlyRootFilesystem == nil || !*context.ReadOnlyRootFilesystem {
		t.Error("forwarder must use a read-only root filesystem")
	}
	if !hasCapability(context.Capabilities.Add, corev1.Capability("DAC_OVERRIDE")) {
		t.Error("forwarder must add DAC_OVERRIDE to write its existing state on the /var/log hostPath")
	}
	if !hasReadOnlyMount(forwarder.VolumeMounts, "varlog", "/var/log") {
		t.Error("forwarder must mount node logs read-only")
	}
	if !hasMount(forwarder.VolumeMounts, "tmp", "/tmp") {
		t.Error("forwarder must mount the tmp volume at /tmp")
	}
}

func assertOpenShiftForwarderSecurity(t *testing.T, podSpec corev1.PodSpec) {
	t.Helper()
	if podSpec.SecurityContext == nil || podSpec.SecurityContext.SELinuxOptions == nil ||
		podSpec.SecurityContext.SELinuxOptions.Type != "spc_t" {
		t.Error("OpenShift forwarder pod must use spc_t to access var_log_t host paths")
	}
	for _, container := range podSpec.Containers {
		expectedGroup := int64(1001)
		if container.Name == "logging-fluentbit-forwarder" {
			expectedGroup = 0
		}
		if container.SecurityContext == nil || container.SecurityContext.RunAsGroup == nil ||
			*container.SecurityContext.RunAsGroup != expectedGroup {
			t.Errorf("OpenShift container %s must use GID %d", container.Name, expectedGroup)
		}
	}
}

func TestAggregatorStatefulSetSecurityContext(t *testing.T) {
	for _, openshift := range []bool{false, true} {
		t.Run(map[bool]string{false: "kubernetes", true: "openshift"}[openshift], func(t *testing.T) {
			statefulSet, err := aggregatorStatefulSet(newAggregatorSecurityLoggingService(openshift))
			if err != nil {
				t.Fatalf("render Fluent Bit aggregator StatefulSet: %v", err)
			}

			podSpec := statefulSet.Spec.Template.Spec
			assertAggregatorPodContext(t, podSpec.SecurityContext)
			assertAggregatorContainers(t, podSpec.Containers)
		})
	}
}

func newAggregatorSecurityLoggingService(openshift bool) *loggingService.LoggingService {
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
	return &loggingService.LoggingService{Spec: loggingService.LoggingServiceSpec{
		OpenshiftDeploy: openshift,
		Fluentbit: &loggingService.Fluentbit{
			DockerImage: "fluent-bit:test",
			Aggregator: &loggingService.FluentbitAggregator{
				DockerImage: "fluent-bit:test",
				Resources:   resources,
				ConfigmapReload: &loggingService.ConfigmapReload{
					DockerImage: "configmap-reload:test",
				},
			},
		},
	}}
}

func assertAggregatorPodContext(t *testing.T, context *corev1.PodSecurityContext) {
	t.Helper()
	if context == nil {
		t.Fatal("aggregator pod security context is missing")
	}
	if context.RunAsNonRoot == nil || !*context.RunAsNonRoot || context.SeccompProfile == nil ||
		context.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("aggregator pod must run as non-root with the RuntimeDefault seccomp profile")
	}
	if context.RunAsUser == nil || *context.RunAsUser != 1001 {
		t.Error("aggregator pod must use UID 1001")
	}
	if context.RunAsGroup == nil || *context.RunAsGroup != 1001 {
		t.Error("aggregator pod must use GID 1001")
	}
}

func assertAggregatorContainers(t *testing.T, containers []corev1.Container) {
	t.Helper()
	for _, container := range containers {
		assertAggregatorContainerContext(t, container)
	}
	if !hasWritableMount(containers[1].VolumeMounts, "storage", "/fluent-bit/storage") {
		t.Error("aggregator must retain its writable storage mount")
	}
}

func assertAggregatorContainerContext(t *testing.T, container corev1.Container) {
	t.Helper()
	context := container.SecurityContext
	if context == nil {
		t.Fatalf("container %s has no security context", container.Name)
	}
	if context.RunAsNonRoot == nil || !*context.RunAsNonRoot || context.AllowPrivilegeEscalation == nil ||
		*context.AllowPrivilegeEscalation || context.ReadOnlyRootFilesystem == nil ||
		!*context.ReadOnlyRootFilesystem || context.Capabilities == nil ||
		!hasCapability(context.Capabilities.Drop, corev1.Capability("ALL")) {
		t.Errorf("container %s must use the hardened non-root security context", container.Name)
	}
	if !hasMount(container.VolumeMounts, "tmp", "/tmp") {
		t.Errorf("container %s must mount the tmp volume", container.Name)
	}
	if context.RunAsUser == nil || *context.RunAsUser != 1001 {
		t.Errorf("container %s must use UID 1001", container.Name)
	}
	if context.RunAsGroup == nil || *context.RunAsGroup != 1001 {
		t.Errorf("container %s must use GID 1001", container.Name)
	}
}

func TestCreateOrUpdateAggregatorStatefulSetUpdatesHardeningFields(t *testing.T) {
	cr := newHAFluentUpdateLoggingService()
	desired, err := aggregatorStatefulSet(cr)
	if err != nil {
		t.Fatalf("render Fluent Bit aggregator StatefulSet: %v", err)
	}
	existing := desired.DeepCopy()
	staleReplicas := int32(1)
	existing.Spec.Replicas = &staleReplicas
	existing.Spec.Template.Spec.SecurityContext = nil
	testScheme := newHAFluentHandlerTestScheme(t)
	testClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(existing).Build()
	reconciler := &HAFluentReconciler{ComponentReconciler: &util.ComponentReconciler{
		Client: testClient,
		Scheme: testScheme,
		Log:    util.Logger("test-ha-fluent-aggregator-update"),
	}}

	if err := reconciler.createOrUpdateAggregatorStatefulSet(cr, desired); err != nil {
		t.Fatalf("update Fluent Bit aggregator StatefulSet: %v", err)
	}
	updated := &appsv1.StatefulSet{}
	key := types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}
	if err := testClient.Get(context.Background(), key, updated); err != nil {
		t.Fatalf("get updated Fluent Bit aggregator StatefulSet: %v", err)
	}
	if updated.Spec.Replicas == nil || *updated.Spec.Replicas != *desired.Spec.Replicas {
		t.Errorf("updated replicas = %v, want %d", updated.Spec.Replicas, *desired.Spec.Replicas)
	}
	if !reflect.DeepEqual(updated.Spec.Template.Spec.SecurityContext, desired.Spec.Template.Spec.SecurityContext) {
		t.Error("updated pod security context does not match the rendered pod security context")
	}
}

func newHAFluentUpdateLoggingService() *loggingService.LoggingService {
	return &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				DockerImage:       "fluent-bit:test",
				PriorityClassName: "system-cluster-critical",
				ConfigmapReload:   &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
				Aggregator: &loggingService.FluentbitAggregator{
					DockerImage:       "fluent-bit:test",
					PriorityClassName: "system-cluster-critical",
					Replicas:          2,
					StartupTimeout:    1,
					ConfigmapReload:   &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
					Resources: &corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				},
			},
		},
	}
}

func hasCapability(capabilities []corev1.Capability, expected corev1.Capability) bool {
	for _, capability := range capabilities {
		if capability == expected {
			return true
		}
	}
	return false
}

func hasWritableMount(mounts []corev1.VolumeMount, name, path string) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path && !mount.ReadOnly {
			return true
		}
	}
	return false
}

func hasReadOnlyMount(mounts []corev1.VolumeMount, name, path string) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path && mount.ReadOnly {
			return true
		}
	}
	return false
}

func hasMount(mounts []corev1.VolumeMount, name, path string) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path {
			return true
		}
	}
	return false
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
	if string(actual.Data["fluent-bit.conf"]) != "new" {
		t.Fatalf("unexpected Secret data: %q", actual.Data["fluent-bit.conf"])
	}
}

// Upgrades from releases that stored the aggregator configuration in a ConfigMap must
// not leave that ConfigMap behind, because nothing reads or deletes it afterwards.
func TestHandleAggregatorConfigSecretRemovesLegacyConfigMap(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := loggingService.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	legacy := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: util.AggregatorFluentbitComponentName, Namespace: "logging"},
		Data:       map[string]string{"fluent-bit.conf": "old"},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(legacy).Build()
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
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				ContainerLogging: true,
				Aggregator:       &loggingService.FluentbitAggregator{Install: true},
			},
		},
	}

	if err := reconciler.handleAggregatorConfigSecret(cr); err != nil {
		t.Fatal(err)
	}

	if err := fakeClient.Get(t.Context(), client.ObjectKeyFromObject(legacy), &corev1.ConfigMap{}); !api_errors.IsNotFound(err) {
		t.Fatalf("the legacy ConfigMap must be deleted, got error %v", err)
	}
	key := types.NamespacedName{Name: util.AggregatorFluentbitComponentName, Namespace: "logging"}
	if err := fakeClient.Get(t.Context(), key, &corev1.Secret{}); err != nil {
		t.Fatalf("the config Secret must be created: %v", err)
	}
}
