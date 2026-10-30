package fluentbit

import (
	"context"
	"reflect"
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	fbbuilder "github.com/Netcracker/qubership-logging-operator/internal/builder/fluentbit"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType
	desired := fbbuilder.DaemonSet(cr.DeepCopy(), defaults.Get())
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
		Defaults:          defaults.Get(),
	}

	if err := reconciler.handleDaemonSet(context.Background(), cr); err != nil {
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
		Defaults:          defaults.Get(),
	}

	if err := reconciler.handleDaemonSet(context.Background(), cr); err != nil {
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

	if err := reconciler.handleDaemonSet(context.Background(), cr); err == nil {
		t.Fatal("expected an error when Fluent Bit configuration is missing")
	}
}

func TestHandleDaemonSetReturnsGetError(t *testing.T) {
	testScheme := runtime.NewScheme()
	if err := loggingService.AddToScheme(testScheme); err != nil {
		t.Fatalf("add LoggingService scheme: %v", err)
	}
	if err := appsv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("add apps scheme: %v", err)
	}
	// The DaemonSet already exists, but reading it fails.
	testClient := fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(interceptor.Funcs{
		Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
			return errors.NewAlreadyExists(appsv1.Resource("daemonsets"), util.FluentbitComponentName)
		},
		Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
			return errors.NewServiceUnavailable("get is unavailable")
		},
	}).Build()
	reconciler := &FluentbitReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: testClient,
			Scheme: testScheme,
			Log:    util.Logger("test-fluentbit-get-error"),
		},
		Defaults: defaults.Get(),
	}
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging"},
		Spec:       loggingService.LoggingServiceSpec{Fluentbit: &loggingService.Fluentbit{DockerImage: "fluent-bit:test"}},
	}

	if err := reconciler.handleDaemonSet(context.Background(), cr); err == nil {
		t.Fatal("expected an error when the existing DaemonSet cannot be read")
	}
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

// assertFluentbitConfigContains reports every rule the rendered configuration is missing.
func assertFluentbitConfigContains(t *testing.T, config, message string, rules ...string) {
	t.Helper()
	for _, rule := range rules {
		if !strings.Contains(config, rule) {
			t.Errorf("%s %q", message, rule)
		}
	}
}

func TestParsedFieldsProtectReservedFields(t *testing.T) {
	configMap, err := fluentbitConfigSecret(&loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{Fluentbit: &loggingService.Fluentbit{}},
	}, util.DynamicParameters{}, outputCredentials{})
	if err != nil {
		t.Fatalf("failed to render Fluent Bit Secret: %v", err)
	}

	enrichConfig := strings.Join(strings.Fields(string(configMap.Data["filter-enrich-fields.conf"])), " ")
	assertFluentbitConfigContains(t, enrichConfig, "missing reserved field rule",
		"Hard_rename namespace parsed_namespace",
		"Hard_rename pod parsed_pod",
		"Hard_rename pod_id parsed_pod_id",
		"Hard_rename container parsed_container",
		"Hard_rename stdio_stream parsed_stdio_stream",
		"Hard_rename source parsed_source",
		"Hard_rename labels parsed_labels",
		"Hard_rename log parsed_log",
		"Hard_rename time parsed_time",
		"Hard_rename level parsed_level",
		"Hard_rename parse_status parsed_parse_status",
		"Hard_rename _parser_input parsed_parser_input",
	)
	if strings.Contains(enrichConfig, "Add_prefix parsed_") {
		t.Error("application fields without protected names must keep their original names")
	}
	streamRenameIndex := strings.Index(enrichConfig, "Hard_rename stream stdio_stream")
	streamAllowlistIndex := strings.Index(enrichConfig, "Allowlist_key stdio_stream")
	if streamRenameIndex < 0 || streamRenameIndex >= streamAllowlistIndex {
		t.Error("the container stream must be renamed before the allowlist is applied")
	}

	sanitizeLiftIndex := strings.Index(enrichConfig, "Operation lift Nested_under log_parsed Add_prefix __payload_")
	sanitizeRemoveIndex := strings.Index(enrichConfig, "Remove __payload__record_metadata")
	sanitizeNestIndex := strings.Index(enrichConfig, "Operation nest Wildcard __payload_* Nest_under log_parsed Remove_prefix __payload_")
	hideIndex := strings.Index(enrichConfig, "Operation nest Wildcard namespace")
	applicationIndex := strings.LastIndex(enrichConfig, "Nested_under log_parsed")
	renameIndex := strings.Index(enrichConfig, "Hard_rename namespace parsed_namespace")
	restoreIndex := strings.LastIndex(enrichConfig, "Nested_under _record_metadata")
	if sanitizeLiftIndex < 0 || sanitizeLiftIndex >= sanitizeRemoveIndex || sanitizeRemoveIndex >= sanitizeNestIndex ||
		sanitizeNestIndex >= hideIndex || hideIndex >= applicationIndex || applicationIndex >= renameIndex ||
		renameIndex >= restoreIndex {
		t.Error("the reserved payload object must be removed before protected fields are hidden and restored")
	}

	// The level filter keeps a source_level that the payload already sets, so the payload value stays in place.
	if strings.Contains(enrichConfig, "Hard_rename source_level") {
		t.Error("an application source_level must keep its name")
	}

	rawValidateConfig := string(configMap.Data["filter-validate.conf"])
	validateConfig := strings.Join(strings.Fields(rawValidateConfig), " ")
	assertFluentbitConfigContains(t, validateConfig, "missing parsed JSON field rule",
		"Rename msg short_message",
		"Rename message short_message",
	)
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
	configSecret, err := fluentbitConfigSecret(&loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{Fluentbit: &loggingService.Fluentbit{}},
	}, util.DynamicParameters{}, outputCredentials{})
	if err != nil {
		t.Fatalf("failed to render Fluent Bit Secret: %v", err)
	}

	genericConfig := strings.Join(strings.Fields(string(configSecret.Data["filter-generic.conf"])), " ")
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

	statusConfig := string(configSecret.Data["filter-validate.conf"]) + string(configSecret.Data["filter-post-generic.conf"])
	if strings.Count(statusConfig, "Key_does_not_exist _parser_input") != 2 {
		t.Error("klog and generic parser success must be detected from the removed _parser_input field")
	}
	if !strings.Contains(string(configSecret.Data["filter-rewrite-tag.conf"]), "Preserve_Key   Off") {
		t.Error("klog parsers must remove original_log after successful parsing")
	}
	for name, content := range configSecret.Data {
		if strings.Contains(string(content), "count_fields") {
			t.Errorf("%s still uses field-count parsing detection", name)
		}
	}
}
