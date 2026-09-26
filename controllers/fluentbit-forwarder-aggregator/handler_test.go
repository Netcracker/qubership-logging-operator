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

func TestAggregatorStatefulSetStorageSizeLimit(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		want       string
	}{
		{name: "default", want: loggingService.FluentbitAggregatorDefaultStorageSizeLimit},
		{name: "configured", configured: "6Gi", want: "6Gi"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cr := newAggregatorSecurityLoggingService(false)
			cr.Spec.Fluentbit.Aggregator.StorageSizeLimit = test.configured
			statefulSet, err := aggregatorStatefulSet(cr)
			if err != nil {
				t.Fatalf("render Fluent Bit aggregator StatefulSet: %v", err)
			}

			var storage *corev1.Volume
			for index := range statefulSet.Spec.Template.Spec.Volumes {
				volume := &statefulSet.Spec.Template.Spec.Volumes[index]
				if volume.Name == "storage" {
					storage = volume
					break
				}
			}
			if storage == nil || storage.EmptyDir == nil || storage.EmptyDir.SizeLimit == nil ||
				storage.EmptyDir.SizeLimit.String() != test.want {
				t.Errorf("unexpected aggregator storage volume: %#v", storage)
			}
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

func TestAggregatorHTTPOutputTimestampConfiguration(t *testing.T) {
	t.Run("uses the root container timestamp", testAggregatorDefaultHTTPTimestamp)
	for name, extraParams := range map[string]string{
		"custom value": "JSON_DATE_KEY custom_timestamp",
		"disabled":     "json_date_key false",
		"empty":        "json_date_key",
		"duplicated":   "json_date_key first\njson_date_key second",
	} {
		t.Run("rejects "+name+" json_date_key for the default URI", func(t *testing.T) {
			testAggregatorRejectsDefaultJSONDateKey(t, extraParams)
		})
	}
	t.Run("preserves custom URI timestamp configuration", testAggregatorCustomHTTPTimestamp)
	t.Run("preserves disabled json_date_key with a custom URI", testAggregatorDisabledCustomJSONDateKey)
}

func newAggregatorHTTPTestLoggingService(uri, extraParams string) *loggingService.LoggingService {
	return &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				Aggregator: &loggingService.FluentbitAggregator{
					Output: &loggingService.OutputFluentbit{
						Http: &loggingService.HttpFluentbit{
							Enabled:     true,
							Uri:         uri,
							ExtraParams: extraParams,
						},
					},
				},
			},
		},
	}
}

func renderAggregatorHTTPOutput(t *testing.T, uri, extraParams string) string {
	t.Helper()
	configMap, err := aggregatorConfigSecret(newAggregatorHTTPTestLoggingService(uri, extraParams), util.DynamicParameters{}, aggregatorOutputCredentials{})
	if err != nil {
		t.Fatalf("failed to render aggregator Secret: %v", err)
	}
	return string(configMap.Data["output-http.conf"])
}

func assertAggregatorOutputContains(t *testing.T, output, expected, message string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Error(message)
	}
}

func assertAggregatorOutputExcludes(t *testing.T, output, unexpected, message string) {
	t.Helper()
	if strings.Contains(output, unexpected) {
		t.Error(message)
	}
}

func testAggregatorDefaultHTTPTimestamp(t *testing.T) {
	output := renderAggregatorHTTPOutput(t, "", "")
	assertAggregatorOutputContains(t, output, "_time_field=time", "expected the default HTTP URI to use the root time field")
	assertAggregatorOutputExcludes(t, output, "ignore_fields=time", "did not expect VictoriaLogs ingestion to ignore its configured time field")
	assertAggregatorOutputContains(t, output, "_stream_fields=namespace,container", "expected the default HTTP URI to use namespace and container stream fields")
	assertAggregatorOutputContains(t, output, "json_date_key          false", "expected HTTP output not to generate a redundant timestamp field")
}

func testAggregatorRejectsDefaultJSONDateKey(t *testing.T, extraParams string) {
	_, err := aggregatorConfigSecret(newAggregatorHTTPTestLoggingService("", extraParams), util.DynamicParameters{}, aggregatorOutputCredentials{})
	if err == nil || !strings.Contains(err.Error(), "must not set json_date_key") {
		t.Fatalf("expected an operator-managed json_date_key error, got: %v", err)
	}
}

func testAggregatorCustomHTTPTimestamp(t *testing.T) {
	const customURI = "/insert/jsonline?_stream_fields=custom&_msg_field=message&_time_field=date"
	output := renderAggregatorHTTPOutput(t, customURI, "json_date_key date")
	assertAggregatorOutputContains(t, output, "uri                    "+customURI, "expected the custom HTTP URI to be preserved")
	assertAggregatorOutputContains(t, output, "json_date_key date", "expected the custom json_date_key to be preserved")
	assertAggregatorOutputExcludes(t, output, "json_date_key          false", "did not expect the operator-managed json_date_key with a custom URI")
	assertAggregatorOutputExcludes(t, output, "ignore_fields=time", "did not expect the operator-managed ignored fields with a custom URI")
}

func testAggregatorDisabledCustomJSONDateKey(t *testing.T) {
	const customURI = "/insert/jsonline?_stream_fields=custom&_msg_field=message"
	output := renderAggregatorHTTPOutput(t, customURI, "json_date_key false")
	assertAggregatorOutputContains(t, output, "json_date_key false", "expected the disabled custom json_date_key to be preserved")
}

func TestParsedFieldsProtectReservedFields(t *testing.T) {
	configMap, err := aggregatorConfigSecret(&loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{Aggregator: &loggingService.FluentbitAggregator{}},
		},
	}, util.DynamicParameters{}, aggregatorOutputCredentials{})
	if err != nil {
		t.Fatalf("failed to render aggregator Secret: %v", err)
	}

	enrichConfig := strings.Join(strings.Fields(string(configMap.Data["filter-enrich-fields.conf"])), " ")
	for _, rule := range []string{
		"Hard_rename namespace parsed_namespace",
		"Hard_rename pod parsed_pod",
		"Hard_rename container parsed_container",
		"Hard_rename source parsed_source",
		"Hard_rename labels parsed_labels",
		"Hard_rename log parsed_log",
		"Hard_rename time parsed_time",
		"Hard_rename level parsed_level",
		"Hard_rename parse_status parsed_parse_status",
		"Hard_rename source_level parsed_source_level",
	} {
		if !strings.Contains(enrichConfig, rule) {
			t.Errorf("missing reserved field rule %q", rule)
		}
	}
	if strings.Contains(enrichConfig, "Add_prefix parsed_") {
		t.Error("application fields without protected names must keep their original names")
	}

	hideIndex := strings.Index(enrichConfig, "Operation nest Wildcard namespace")
	applicationIndex := strings.Index(enrichConfig, "Nested_under log_parsed")
	renameIndex := strings.Index(enrichConfig, "Hard_rename namespace parsed_namespace")
	restoreIndex := strings.LastIndex(enrichConfig, "Nested_under _record_metadata")
	if hideIndex < 0 || hideIndex >= applicationIndex || applicationIndex >= renameIndex || renameIndex >= restoreIndex {
		t.Error("protected fields must be hidden, application fields extracted and renamed, then protected fields restored")
	}

	levelConfig := strings.Join(strings.Fields(string(configMap.Data["filter-nonsupported-levels.conf"])), " ")
	if !strings.Contains(levelConfig, "Rename parsed_source_level source_level") {
		t.Error("source_level must be restored without overwriting the normalized value")
	}

	rawValidateConfig := string(configMap.Data["filter-validate.conf"])
	validateConfig := strings.Join(strings.Fields(rawValidateConfig), " ")
	for _, rule := range []string{
		"Rename msg short_message",
		"Rename message short_message",
	} {
		if !strings.Contains(validateConfig, rule) {
			t.Errorf("missing parsed JSON field rule %q", rule)
		}
	}
	if strings.Contains(validateConfig, "Rename parsed_msg short_message") ||
		strings.Contains(validateConfig, "Rename parsed_message short_message") {
		t.Error("msg and message are lifted from log_parsed without a parsed_ prefix")
	}

	// VictoriaLogs reads the event time from the root time field. The logfmt parser runs with
	// Reserve_Data On and cannot overwrite that field, so nothing may move it to log_time.
	postGenericConfig := strings.Join(strings.Fields(string(configMap.Data["filter-post-generic.conf"])), " ")
	if strings.Contains(postGenericConfig, "Rename time log_time") {
		t.Error("logfmt records must keep the container timestamp in the time field")
	}

	// Annotation-based parsers never reach the JSON branch, so a conditional restore leaves their
	// level in parsed_level and the normalizer falls back to info.
	levelRestore, found := filterBlockContaining(rawValidateConfig, "Rename parsed_level level")
	if !found {
		t.Error("filter-validate.conf must restore the application level from parsed_level")
	} else if strings.Contains(levelRestore, "Condition") {
		t.Error("the parsed_level restore must apply to every parsed format, not only JSON")
	}
}

// filterBlockContaining returns the [FILTER] section holding the given rule, with the rule written
// as single-spaced text. The second result reports whether any section holds it.
func filterBlockContaining(config, rule string) (string, bool) {
	for _, block := range strings.Split(config, "[FILTER]") {
		if strings.Contains(strings.Join(strings.Fields(block), " "), rule) {
			return block, true
		}
	}
	return "", false
}

func TestParserSuccessUsesPreserveKeyOff(t *testing.T) {
	configMap, err := aggregatorConfigMap(&loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{Aggregator: &loggingService.FluentbitAggregator{}},
		},
	}, util.DynamicParameters{})
	if err != nil {
		t.Fatalf("failed to render aggregator ConfigMap: %v", err)
	}

	genericConfig := strings.Join(strings.Fields(configMap.Data["filter-generic.conf"]), " ")
	for _, expected := range []string{
		"Copy log _parser_input",
		"Key_Name _parser_input",
		"Preserve_Key Off",
	} {
		if !strings.Contains(genericConfig, expected) {
			t.Errorf("generic parser pipeline is missing %q", expected)
		}
	}
	if strings.Contains(genericConfig, "Preserve_Key On") {
		t.Error("generic parsers must remove original_log after successful parsing")
	}

	statusConfig := configMap.Data["filter-validate.conf"] + configMap.Data["filter-post-generic.conf"]
	if strings.Count(statusConfig, "Key_does_not_exist _parser_input") != 2 {
		t.Error("klog and generic parser success must be detected from the removed _parser_input field")
	}
	if !strings.Contains(configMap.Data["filter-enrich-fields.conf"], "Preserve_Key    Off") {
		t.Error("klog parsers must remove original_log after successful parsing")
	}
	for name, content := range configMap.Data {
		if strings.Contains(content, "count_fields") {
			t.Errorf("%s still uses field-count parsing detection", name)
		}
	}
}
