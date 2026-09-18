package fluentd

import (
	"reflect"
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFluentdDaemonSetSecurityContext(t *testing.T) {
	for _, privileged := range []bool{false, true} {
		t.Run(map[bool]string{false: "unprivileged", true: "privileged"}[privileged], func(t *testing.T) {
			cr := &loggingService.LoggingService{
				Spec: loggingService.LoggingServiceSpec{
					Fluentd: &loggingService.Fluentd{
						DockerImage:               "fluentd:test",
						SecurityContextPrivileged: privileged,
						ConfigmapReload: &loggingService.ConfigmapReload{
							DockerImage: "configmap-reload:test",
						},
					},
				},
			}

			daemonSet, err := fluentdDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
			if err != nil {
				t.Fatalf("render Fluentd DaemonSet: %v", err)
			}

			podSpec := daemonSet.Spec.Template.Spec
			assertFluentdPodContext(t, podSpec.SecurityContext)
			assertFluentdReloaderContext(t, podSpec.Containers[0].SecurityContext)
			assertFluentdContainerSecurity(t, podSpec.Containers[1], privileged)
			assertContainersMountTmp(t, podSpec.Containers)
		})
	}
}

func assertFluentdPodContext(t *testing.T, context *corev1.PodSecurityContext) {
	t.Helper()
	if context == nil || context.RunAsUser == nil || *context.RunAsUser != 0 || context.RunAsNonRoot == nil ||
		*context.RunAsNonRoot || context.RunAsGroup == nil || *context.RunAsGroup != 0 ||
		context.SeccompProfile == nil || context.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("Fluentd pod must run as root with the RuntimeDefault seccomp profile")
	}
}

func assertFluentdReloaderContext(t *testing.T, context *corev1.SecurityContext) {
	t.Helper()
	if context == nil || context.RunAsUser == nil || *context.RunAsUser != 1001 || context.RunAsNonRoot == nil ||
		!*context.RunAsNonRoot || context.RunAsGroup == nil || *context.RunAsGroup != 1001 || !isHardened(context) {
		t.Error("configmap-reload must use the hardened non-root security context")
	}
}

func assertFluentdContainerSecurity(t *testing.T, container corev1.Container, privileged bool) {
	t.Helper()
	context := container.SecurityContext
	if context == nil {
		t.Fatal("Fluentd security context is missing")
	}
	if context.RunAsUser == nil || *context.RunAsUser != 0 || context.RunAsNonRoot == nil || *context.RunAsNonRoot ||
		context.RunAsGroup == nil || *context.RunAsGroup != 0 || context.Privileged == nil ||
		*context.Privileged != privileged || context.ReadOnlyRootFilesystem == nil ||
		!*context.ReadOnlyRootFilesystem {
		t.Error("Fluentd must retain root access and a read-only root filesystem")
	}
	if !privileged && !isHardened(context) {
		t.Error("unprivileged Fluentd must disable escalation and drop all capabilities")
	}
	if !hasWritableMount(container.VolumeMounts, "varlog", "/var/log") {
		t.Error("Fluentd must mount node logs read-write to persist position files")
	}
}

func assertContainersMountTmp(t *testing.T, containers []corev1.Container) {
	t.Helper()
	for _, container := range containers {
		if !hasMount(container.VolumeMounts, "tmp", "/tmp") {
			t.Errorf("container %s must mount the tmp volume", container.Name)
		}
	}
}

func TestFluentdConfigSecretUsesLegacyPositionFiles(t *testing.T) {
	for _, systemLogType := range []string{"varlogmessages", "varlogsyslog", "systemd"} {
		t.Run(systemLogType, func(t *testing.T) {
			cr := &loggingService.LoggingService{
				Spec: loggingService.LoggingServiceSpec{
					Fluentd: &loggingService.Fluentd{
						SystemLogging:             true,
						SystemLogType:             systemLogType,
						SystemAuditLogging:        true,
						KubeAuditLogging:          true,
						KubeApiserverAuditLogging: true,
						ContainerLogging:          true,
					},
				},
			}

			secret, err := fluentdConfigSecret(cr,
				util.DynamicParameters{ContainerRuntimeType: "containerd"}, outputCredentials{})
			if err != nil {
				t.Fatalf("render Fluentd configuration Secret: %v", err)
			}
			assertLegacyPositionFiles(t, secret.Data, systemLogType)
		})
	}
}

func assertLegacyPositionFiles(t *testing.T, data map[string][]byte, systemLogType string) {
	t.Helper()
	expected := expectedLegacyPositionFiles(systemLogType)
	for _, file := range positionFiles(data) {
		if _, found := expected[file.path]; !found {
			t.Errorf("Secret entry %s uses unexpected position file %q", file.secretEntry, file.path)
		}
		delete(expected, file.path)
	}
	for path := range expected {
		t.Errorf("Secret does not use legacy position file %q", path)
	}
}

func expectedLegacyPositionFiles(systemLogType string) map[string]struct{} {
	return map[string]struct{}{
		"/var/log/es-containers.log.pos":        {},
		"/var/log/audit/audit.log.pos":          {},
		"/var/log/kube-audit.log.pos":           {},
		"/var/log/kube-apiserver.log.pos":       {},
		"/var/log/kube-apiserver-audit.log.pos": {},
		"/var/log/openshift-apiserver.log.pos":  {},
		map[string]string{
			"varlogmessages": "/var/log/messages.pos",
			"varlogsyslog":   "/var/log/syslog.pos",
			"systemd":        "/var/log/journal.pos",
		}[systemLogType]: {},
	}
}

type positionFile struct {
	secretEntry string
	path        string
}

func positionFiles(data map[string][]byte) []positionFile {
	var files []positionFile
	for name, content := range data {
		for _, line := range strings.Split(string(content), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] == "pos_file" {
				files = append(files, positionFile{secretEntry: name, path: fields[1]})
			}
		}
	}
	return files
}

func TestFluentdOpenShiftSecurityContext(t *testing.T) {
	cr := &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			OpenshiftDeploy: true,
			Fluentd: &loggingService.Fluentd{
				DockerImage: "fluentd:test",
				ConfigmapReload: &loggingService.ConfigmapReload{
					DockerImage: "configmap-reload:test",
				},
			},
		},
	}
	daemonSet, err := fluentdDaemonSet(cr, util.DynamicParameters{ContainerRuntimeType: "containerd"})
	if err != nil {
		t.Fatalf("render OpenShift Fluentd DaemonSet: %v", err)
	}
	podContext := daemonSet.Spec.Template.Spec.SecurityContext
	if podContext == nil || podContext.SELinuxOptions == nil || podContext.SELinuxOptions.Type != "spc_t" {
		t.Error("OpenShift Fluentd pod must use spc_t to access var_log_t host paths")
	}
}

func TestHandleDaemonSetUpdatesTheCompletePodSpec(t *testing.T) {
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentd: &loggingService.Fluentd{
				DockerImage:       "fluentd:test",
				PriorityClassName: "system-cluster-critical",
				ConfigmapReload:   &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
			},
		},
	}
	dynamicParameters := util.DynamicParameters{ContainerRuntimeType: "containerd"}
	desired, err := fluentdDaemonSet(cr, dynamicParameters)
	if err != nil {
		t.Fatalf("render FluentD DaemonSet: %v", err)
	}
	existing := desired.DeepCopy()
	existing.Spec.Template.Spec = corev1.PodSpec{PriorityClassName: "stale-priority"}

	testScheme := runtime.NewScheme()
	if err := loggingService.AddToScheme(testScheme); err != nil {
		t.Fatalf("add LoggingService scheme: %v", err)
	}
	if err := appsv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("add apps scheme: %v", err)
	}
	testClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(existing).Build()
	reconciler := &FluentdReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: testClient,
			Scheme: testScheme,
			Log:    util.Logger("test-fluentd-update"),
		},
		DynamicParameters: dynamicParameters,
	}

	if err := reconciler.handleDaemonSet(cr); err != nil {
		t.Fatalf("update FluentD DaemonSet: %v", err)
	}
	updated := &appsv1.DaemonSet{}
	if err := testClient.Get(t.Context(), client.ObjectKeyFromObject(desired), updated); err != nil {
		t.Fatalf("get updated FluentD DaemonSet: %v", err)
	}
	if !reflect.DeepEqual(updated.Spec.Template.Spec, desired.Spec.Template.Spec) {
		t.Errorf("updated pod spec does not match the rendered pod spec")
	}
}

func isHardened(context *corev1.SecurityContext) bool {
	return context.AllowPrivilegeEscalation != nil && !*context.AllowPrivilegeEscalation &&
		context.ReadOnlyRootFilesystem != nil && *context.ReadOnlyRootFilesystem &&
		context.Capabilities != nil && hasCapability(context.Capabilities.Drop, corev1.Capability("ALL"))
}

func hasCapability(capabilities []corev1.Capability, expected corev1.Capability) bool {
	for _, capability := range capabilities {
		if capability == expected {
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

func hasWritableMount(mounts []corev1.VolumeMount, name, path string) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path && !mount.ReadOnly {
			return true
		}
	}
	return false
}

func TestResolveOutputCredentials(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "output-auth", Namespace: "logging"},
		Data: map[string][]byte{
			"username": []byte("fluentd-user"),
			"password": []byte("fluentd-password"),
			"token":    []byte("fluentd-token"),
		},
	}
	reconciler := &FluentdReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build(),
			Log:    util.Logger("test-fluentd"),
		},
	}
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentd: &loggingService.Fluentd{
				Output: &loggingService.OutputFluentd{
					Http: &loggingService.HttpFluentd{
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
	if credentials.Http.Token != "fluentd-token" ||
		credentials.Http.User != "fluentd-user" ||
		credentials.Http.Password != "fluentd-password" {
		t.Fatalf("unexpected resolved credentials: %#v", credentials.Http)
	}

	configSecret, err := fluentdConfigSecret(cr, util.DynamicParameters{}, credentials)
	if err != nil {
		t.Fatal(err)
	}
	httpOutput := string(configSecret.Data["output-http.conf"])
	for _, expected := range []string{"fluentd-user", "fluentd-password", "Bearer fluentd-token"} {
		if !strings.Contains(httpOutput, expected) {
			t.Errorf("generated HTTP output does not contain %q", expected)
		}
	}
}

func TestCreateOrUpdateConfigSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := loggingService.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-fluentd", Namespace: "logging"},
		Data:       map[string][]byte{"fluent.conf": []byte("old")},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	reconciler := &FluentdReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client: fakeClient,
			Scheme: scheme,
			Log:    util.Logger("test-fluentd"),
		},
	}
	cr := &loggingService.LoggingService{
		TypeMeta:   metav1.TypeMeta{APIVersion: loggingService.GroupVersion.String(), Kind: "LoggingService"},
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging", UID: "test-uid"},
	}
	desired := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-fluentd", Namespace: "logging"},
		Data:       map[string][]byte{"fluent.conf": []byte("new")},
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
	if string(actual.Data["fluent.conf"]) != "new" {
		t.Fatalf("unexpected Secret data: %q", actual.Data["fluent.conf"])
	}
}

func TestFluentdConfigSecretHTTPHeadersWithToken(t *testing.T) {
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentd: &loggingService.Fluentd{
				Output: &loggingService.OutputFluentd{
					Http: &loggingService.HttpFluentd{
						Enabled: true,
						Headers: map[string]string{"X-Scope-OrgID": "tenant-1"},
						Auth: &loggingService.Auth{
							Token: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "output-auth"},
								Key:                  "token",
							},
						},
					},
				},
			},
		},
	}
	credentials := outputCredentials{Http: util.AuthValues{Token: "fluentd-token"}}

	secret, err := fluentdConfigSecret(cr, util.DynamicParameters{}, credentials)
	if err != nil {
		t.Fatalf("custom headers combined with a bearer token must render: %v", err)
	}
	httpOutput := string(secret.Data["output-http.conf"])
	for _, expected := range []string{`"X-Scope-OrgID":"tenant-1"`, `"Authorization":"Bearer fluentd-token"`} {
		if !strings.Contains(httpOutput, expected) {
			t.Errorf("generated HTTP output does not contain %q", expected)
		}
	}
}

func TestFluentdConfigSecretCredentialsAreNotInterpolated(t *testing.T) {
	selector := func(key string) *corev1.SecretKeySelector {
		return &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "output-auth"},
			Key:                  key,
		}
	}
	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentd: &loggingService.Fluentd{
				Output: &loggingService.OutputFluentd{
					Loki: &loggingService.LokiFluentd{
						Enabled: true,
						Auth: &loggingService.Auth{
							User:     selector("username"),
							Password: selector("password"),
						},
					},
					Http: &loggingService.HttpFluentd{
						Enabled: true,
						Auth: &loggingService.Auth{
							User:     selector("username"),
							Password: selector("password"),
						},
					},
				},
			},
		},
	}
	// A password that FluentD would evaluate as embedded Ruby inside a double-quoted string.
	password := `p#{exec("id")}a`
	credentials := outputCredentials{
		Loki: util.AuthValues{User: "loki-user", Password: password},
		Http: util.AuthValues{User: "http-user", Password: password},
	}

	secret, err := fluentdConfigSecret(cr, util.DynamicParameters{}, credentials)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"output-loki.conf", "output-http.conf"} {
		rendered := string(secret.Data[file])
		if !strings.Contains(rendered, `password   'p#{exec("id")}a'`) &&
			!strings.Contains(rendered, `password 'p#{exec("id")}a'`) {
			t.Errorf("%s must quote the password with single quotes, got:\n%s", file, rendered)
		}
		if strings.Contains(rendered, `password "`) || strings.Contains(rendered, `password   "`) {
			t.Errorf("%s must not render the password as a double-quoted string, got:\n%s", file, rendered)
		}
	}
}
