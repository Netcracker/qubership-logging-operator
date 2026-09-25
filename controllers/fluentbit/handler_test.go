package fluentbit

import (
	"context"
	"reflect"
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func TestFluentbitDaemonSetHardeningExceptions(t *testing.T) {
	for _, privileged := range []bool{false, true} {
		t.Run(map[bool]string{false: "unprivileged", true: "privileged"}[privileged], func(t *testing.T) {
			cr := &loggingService.LoggingService{
				Spec: loggingService.LoggingServiceSpec{
					Fluentbit: &loggingService.Fluentbit{
						DockerImage:               "fluent-bit:test",
						ConfigmapReload:           &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
						SecurityContextPrivileged: privileged,
					},
				},
			}

			daemonSet, err := fluentbitDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
			if err != nil {
				t.Fatalf("render Fluent Bit DaemonSet: %v", err)
			}

			podSpec := daemonSet.Spec.Template.Spec
			assertRuntimeDefaultPodContext(t, podSpec.SecurityContext)
			assertFluentbitReloaderContext(t, podSpec.Containers[0].SecurityContext)
			assertFluentbitCollectorSecurity(t, podSpec.Containers[1], privileged)
		})
	}
}

func assertRuntimeDefaultPodContext(t *testing.T, context *corev1.PodSecurityContext) {
	t.Helper()
	if context == nil || context.SeccompProfile == nil ||
		context.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("pod must use the RuntimeDefault seccomp profile")
	}
}

func assertFluentbitReloaderContext(t *testing.T, context *corev1.SecurityContext) {
	t.Helper()
	if context == nil || context.ReadOnlyRootFilesystem == nil || !*context.ReadOnlyRootFilesystem ||
		context.RunAsNonRoot == nil || !*context.RunAsNonRoot || context.RunAsGroup == nil ||
		*context.RunAsGroup != 1001 {
		t.Error("configmap-reload must run as non-root with a read-only root filesystem")
	}
}

func assertFluentbitCollectorSecurity(t *testing.T, collector corev1.Container, privileged bool) {
	t.Helper()
	if collector.SecurityContext == nil {
		t.Fatal("collector security context is missing")
	}
	assertRootCollectorContext(t, collector.SecurityContext)
	if collector.SecurityContext.Privileged == nil || *collector.SecurityContext.Privileged != privileged {
		t.Errorf("collector privileged setting = %v, want %v", collector.SecurityContext.Privileged, privileged)
	}
	if !privileged && !hasCapability(collector.SecurityContext.Capabilities.Add, corev1.Capability("DAC_OVERRIDE")) {
		t.Error("unprivileged collector must add DAC_OVERRIDE to write its existing state on the /var/log hostPath")
	}
	if !hasReadOnlyMount(collector.VolumeMounts, "varlog", "/var/log") {
		t.Error("collector must mount node logs read-only")
	}
	if !hasMount(collector.VolumeMounts, "tmp", "/tmp") {
		t.Error("collector must mount the tmp volume at /tmp")
	}
}

func assertRootCollectorContext(t *testing.T, context *corev1.SecurityContext) {
	t.Helper()
	if context.RunAsUser == nil || *context.RunAsUser != 0 || context.RunAsNonRoot == nil ||
		*context.RunAsNonRoot || context.RunAsGroup == nil || *context.RunAsGroup != 0 {
		t.Error("collector must run as root to access node logs and its existing state under /var/log")
	}
	if context.ReadOnlyRootFilesystem == nil || !*context.ReadOnlyRootFilesystem {
		t.Error("collector must use a read-only root filesystem")
	}
}

func TestFluentbitOpenShiftSecurityContext(t *testing.T) {
	cr := &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			OpenshiftDeploy: true,
			Fluentbit: &loggingService.Fluentbit{
				DockerImage:     "fluent-bit:test",
				ConfigmapReload: &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
			},
		},
	}

	daemonSet, err := fluentbitDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
	if err != nil {
		t.Fatalf("render OpenShift Fluent Bit DaemonSet: %v", err)
	}
	podContext := daemonSet.Spec.Template.Spec.SecurityContext
	if podContext == nil || podContext.SELinuxOptions == nil || podContext.SELinuxOptions.Type != "spc_t" {
		t.Error("OpenShift collector pod must use spc_t to access var_log_t host paths")
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

func newTestFluentbitReconciler() *FluentbitReconciler {
	return &FluentbitReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Log: util.Logger("test-fluentbit"),
		},
	}
}

// Verifies that resolveOutputCredentials correctly resolves Auth references
// (SecretKeySelector for username/password/token) into actual values from a Kubernetes
// Secret, and that these values are inlined into the rendered config Secret
// (output-http.conf) instead of ${HTTP_USERNAME}-style placeholders.
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
		t.Fatalf("unexpected resolved credentials: %#v", credentials.Http)
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
}

// Regression test for the fix that sets ResourceVersion on the desired Secret before
// updating: without it, UpdateResource fails against a real/fake client because the
// object it's given has no ResourceVersion set.
func TestCreateOrUpdateConfigSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := loggingService.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-fluentbit", Namespace: "logging"},
		Data:       map[string][]byte{"fluent-bit.conf": []byte("old")},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	reconciler := &FluentbitReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: fakeClient,
			Scheme: scheme,
			Log:    util.Logger("test-fluentbit"),
		},
	}
	cr := &loggingService.LoggingService{
		TypeMeta:   metav1.TypeMeta{APIVersion: loggingService.GroupVersion.String(), Kind: "LoggingService"},
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging", UID: "test-uid"},
	}
	desired := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-fluentbit", Namespace: "logging"},
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

// Upgrades from releases that stored the Fluent Bit configuration in a ConfigMap must
// not leave that ConfigMap behind, because nothing reads or deletes it afterwards.
func TestHandleConfigSecretRemovesLegacyConfigMap(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := loggingService.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	legacy := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: util.FluentbitComponentName, Namespace: "logging"},
		Data:       map[string]string{"fluent-bit.conf": "old"},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(legacy).Build()
	reconciler := &FluentbitReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: fakeClient,
			Scheme: scheme,
			Log:    util.Logger("test-fluentbit"),
		},
	}
	cr := &loggingService.LoggingService{
		TypeMeta:   metav1.TypeMeta{APIVersion: loggingService.GroupVersion.String(), Kind: "LoggingService"},
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging", UID: "test-uid"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{ContainerLogging: true},
		},
	}

	if err := reconciler.handleConfigSecret(cr); err != nil {
		t.Fatal(err)
	}

	if err := fakeClient.Get(t.Context(), client.ObjectKeyFromObject(legacy), &corev1.ConfigMap{}); !errors.IsNotFound(err) {
		t.Fatalf("the legacy ConfigMap must be deleted, got error %v", err)
	}
	configSecret := &corev1.Secret{}
	key := types.NamespacedName{Name: util.FluentbitComponentName, Namespace: "logging"}
	if err := fakeClient.Get(t.Context(), key, configSecret); err != nil {
		t.Fatalf("the config Secret must be created: %v", err)
	}
	if len(configSecret.Data["fluent-bit.conf"]) == 0 {
		t.Error("the config Secret must carry the rendered configuration")
	}
}

func TestFluentbitHTTPOutputTimestampConfiguration(t *testing.T) {
	t.Run("uses the root container timestamp", testFluentbitDefaultHTTPTimestamp)
	for name, extraParams := range map[string]string{
		"custom value": "JSON_DATE_KEY custom_timestamp",
		"disabled":     "json_date_key false",
		"empty":        "json_date_key",
		"duplicated":   "json_date_key first\njson_date_key second",
	} {
		t.Run("rejects "+name+" json_date_key for the default URI", func(t *testing.T) {
			testFluentbitRejectsDefaultJSONDateKey(t, extraParams)
		})
	}
	t.Run("preserves custom URI timestamp configuration", testFluentbitCustomHTTPTimestamp)
	t.Run("preserves disabled json_date_key with a custom URI", testFluentbitDisabledCustomJSONDateKey)
}

func newFluentbitHTTPTestLoggingService(uri, extraParams string) *loggingService.LoggingService {
	return &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				Output: &loggingService.OutputFluentbit{
					Http: &loggingService.HttpFluentbit{
						Enabled:     true,
						Uri:         uri,
						ExtraParams: extraParams,
					},
				},
			},
		},
	}
}

func renderFluentbitHTTPOutput(t *testing.T, uri, extraParams string) string {
	t.Helper()
	configMap, err := fluentbitConfigSecret(newFluentbitHTTPTestLoggingService(uri, extraParams), util.DynamicParameters{}, outputCredentials{})
	if err != nil {
		t.Fatalf("failed to render Fluent Bit Secret: %v", err)
	}
	return string(configMap.Data["output-http.conf"])
}

func assertFluentbitOutputContains(t *testing.T, output, expected, message string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Error(message)
	}
}

func assertFluentbitOutputExcludes(t *testing.T, output, unexpected, message string) {
	t.Helper()
	if strings.Contains(output, unexpected) {
		t.Error(message)
	}
}

func testFluentbitDefaultHTTPTimestamp(t *testing.T) {
	output := renderFluentbitHTTPOutput(t, "", "")
	assertFluentbitOutputContains(t, output, "_time_field=time", "expected the default HTTP URI to use the root time field")
	assertFluentbitOutputExcludes(t, output, "ignore_fields=time", "did not expect VictoriaLogs ingestion to ignore its configured time field")
	assertFluentbitOutputContains(t, output, "_stream_fields=namespace,container", "expected the default HTTP URI to use namespace and container stream fields")
	assertFluentbitOutputContains(t, output, "json_date_key          false", "expected HTTP output not to generate a redundant timestamp field")
}

func testFluentbitRejectsDefaultJSONDateKey(t *testing.T, extraParams string) {
	_, err := fluentbitConfigSecret(newFluentbitHTTPTestLoggingService("", extraParams), util.DynamicParameters{}, outputCredentials{})
	if err == nil || !strings.Contains(err.Error(), "must not set json_date_key") {
		t.Fatalf("expected an operator-managed json_date_key error, got: %v", err)
	}
}

func testFluentbitCustomHTTPTimestamp(t *testing.T) {
	const customURI = "/insert/jsonline?_stream_fields=custom&_msg_field=message&_time_field=date"
	output := renderFluentbitHTTPOutput(t, customURI, "json_date_key date")
	assertFluentbitOutputContains(t, output, "uri                    "+customURI, "expected the custom HTTP URI to be preserved")
	assertFluentbitOutputContains(t, output, "json_date_key date", "expected the custom json_date_key to be preserved")
	assertFluentbitOutputExcludes(t, output, "json_date_key          false", "did not expect the operator-managed json_date_key with a custom URI")
	assertFluentbitOutputExcludes(t, output, "ignore_fields=time", "did not expect the operator-managed ignored fields with a custom URI")
}

func testFluentbitDisabledCustomJSONDateKey(t *testing.T) {
	const customURI = "/insert/jsonline?_stream_fields=custom&_msg_field=message"
	output := renderFluentbitHTTPOutput(t, customURI, "json_date_key false")
	assertFluentbitOutputContains(t, output, "json_date_key false", "expected the disabled custom json_date_key to be preserved")
}

func TestParsedFieldsProtectReservedFields(t *testing.T) {
	configMap, err := fluentbitConfigSecret(&loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{Fluentbit: &loggingService.Fluentbit{}},
	}, util.DynamicParameters{}, outputCredentials{})
	if err != nil {
		t.Fatalf("failed to render Fluent Bit Secret: %v", err)
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

	validateConfig := strings.Join(strings.Fields(string(configMap.Data["filter-validate.conf"])), " ")
	for _, rule := range []string{
		"Rename msg short_message",
		"Rename message short_message",
		"Rename parsed_level level",
	} {
		if !strings.Contains(validateConfig, rule) {
			t.Errorf("missing parsed JSON field rule %q", rule)
		}
	}
	if strings.Contains(validateConfig, "Rename parsed_msg short_message") ||
		strings.Contains(validateConfig, "Rename parsed_message short_message") {
		t.Error("msg and message are lifted from log_parsed without a parsed_ prefix")
	}
}
