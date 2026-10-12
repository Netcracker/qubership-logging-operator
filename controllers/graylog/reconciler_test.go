package graylog

import (
	"context"
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

func TestGraylogWorkloadsUseHardenedApplicationContainers(t *testing.T) {
	_, cr := newGraylogTestReconciler(t)
	cr.Spec.Graylog.DockerImage = "docker.io/graylog/graylog:5.2.12"
	cr.Spec.Graylog.MongoDBImage = "docker.io/mongo:5.0.33"
	cr.Spec.Graylog.InitSetupImage = "docker.io/alpine:3.23.4"
	cr.Spec.Graylog.InitContainerDockerImage = "docker.io/alpine:3.23.4"

	assertGraylogKubernetesWorkload(t, cr)
	cr.Spec.OpenshiftDeploy = true
	assertGraylogOpenShiftWorkload(t, cr)
}

func TestGraylogStatefulsetIgnoresDeprecatedMongoDBUpgrade(t *testing.T) {
	_, cr := newGraylogTestReconciler(t)
	cr.Spec.Graylog.DockerImage = "docker.io/graylog/graylog:5.2.12"
	cr.Spec.Graylog.MongoDBImage = "docker.io/mongo:5.0.33"
	cr.Spec.Graylog.InitSetupImage = "docker.io/alpine:3.23.4"

	want, err := graylogStatefulset(cr)
	if err != nil {
		t.Fatalf("render Graylog StatefulSet: %v", err)
	}
	cr.Spec.Graylog.MongoDBUpgrade = &loggingService.MongoDBUpgrade{
		MongoDBImage40: "docker.io/mongo:4.0.28",
		MongoDBImage42: "docker.io/mongo:4.2.22",
		MongoDBImage44: "docker.io/mongo:4.4.17",
	}
	got, err := graylogStatefulset(cr)
	if err != nil {
		t.Fatalf("render Graylog StatefulSet with mongoDBUpgrade: %v", err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatal("mongoDBUpgrade must not change the Graylog StatefulSet")
	}
}

func TestGraylogDownloadPluginsTargetsGraylog5ForAnyImage(t *testing.T) {
	for _, image := range []string{"docker.io/graylog/graylog:5.2.12", "registry.example.com/graylog:custom"} {
		t.Run(image, func(t *testing.T) {
			_, cr := newGraylogTestReconciler(t)
			cr.Spec.Graylog.DockerImage = image
			cr.Spec.Graylog.MongoDBImage = "docker.io/mongo:5.0.33"
			cr.Spec.Graylog.InitSetupImage = "docker.io/alpine:3.23.4"
			cr.Spec.Graylog.InitContainerDockerImage = "docker.io/alpine:3.23.4"

			statefulSet, err := graylogStatefulset(cr)
			if err != nil {
				t.Fatalf("render Graylog StatefulSet: %v", err)
			}
			downloadPlugins := statefulSet.Spec.Template.Spec.InitContainers[1]
			for _, env := range downloadPlugins.Env {
				if env.Name == "GRAYLOG_VERSION" {
					if env.Value != "5" {
						t.Fatalf("GRAYLOG_VERSION = %q, want 5", env.Value)
					}
					return
				}
			}
			t.Fatal("download-plugins must set GRAYLOG_VERSION")
		})
	}
}

func assertGraylogKubernetesWorkload(t *testing.T, cr *loggingService.LoggingService) {
	t.Helper()
	statefulSet, err := graylogStatefulset(cr)
	if err != nil {
		t.Fatalf("render Graylog StatefulSet: %v", err)
	}
	if statefulSet.Spec.Template.Spec.SecurityContext == nil ||
		statefulSet.Spec.Template.Spec.SecurityContext.SeccompProfile == nil ||
		statefulSet.Spec.Template.Spec.SecurityContext.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Fatal("Graylog pod must use the RuntimeDefault seccomp profile")
	}
	mongo := statefulSet.Spec.Template.Spec.Containers[0]
	graylog := statefulSet.Spec.Template.Spec.Containers[1]
	assertHardenedContainer(t, mongo, nil)
	assertHardenedContainer(t, graylog, []corev1.Capability{"NET_BIND_SERVICE"})
	assertRunAsGroup(t, mongo, 1001)
	assertRunAsGroup(t, graylog, 1100)
	downloadPlugins := statefulSet.Spec.Template.Spec.InitContainers[1]
	if downloadPlugins.Name != "download-plugins" {
		t.Fatalf("second init container = %q, want download-plugins", downloadPlugins.Name)
	}
	assertRunAsUser(t, downloadPlugins, 1001)
	assertBoundedEmptyDirs(t, statefulSet.Spec.Template.Spec.Volumes)
	assertGraylogDataPermissions(t, statefulSet, false)
}

func assertGraylogOpenShiftWorkload(t *testing.T, cr *loggingService.LoggingService) {
	t.Helper()
	openShiftStatefulSet, err := graylogStatefulset(cr)
	if err != nil {
		t.Fatalf("render OpenShift Graylog StatefulSet: %v", err)
	}
	openShiftMongo := openShiftStatefulSet.Spec.Template.Spec.Containers[0]
	openShiftGraylog := openShiftStatefulSet.Spec.Template.Spec.Containers[1]
	assertRunAsUser(t, openShiftMongo, 1001)
	assertRunAsUser(t, openShiftGraylog, 1100)
	assertRunAsGroup(t, openShiftMongo, 1001)
	assertRunAsGroup(t, openShiftGraylog, 1100)
	assertGraylogDataPermissions(t, openShiftStatefulSet, true)
}

func TestGraylogDoesNotMountHTTPSecretsWhenTLSIsDisabled(t *testing.T) {
	_, cr := newGraylogTestReconciler(t)
	cr.Spec.Graylog.DockerImage = "docker.io/graylog/graylog:5.2.12"
	cr.Spec.Graylog.MongoDBImage = "docker.io/mongo:5.0.33"
	cr.Spec.Graylog.InitSetupImage = "docker.io/alpine:3.23.4"
	cr.Spec.Graylog.TLS = &loggingService.GraylogTLS{
		HTTP: &loggingService.HTTPGraylogTLS{Enabled: false},
	}

	statefulSet, err := graylogStatefulset(cr)
	if err != nil {
		t.Fatalf("render Graylog StatefulSet: %v", err)
	}
	command := statefulSet.Spec.Template.Spec.Containers[1].Command
	for _, argument := range command {
		if strings.Contains(argument, "keytool") || strings.Contains(argument, "tls.crt") {
			t.Fatalf("disabled HTTP TLS generated certificate setup command: %q", argument)
		}
	}
}

func assertHardenedContainer(t *testing.T, container corev1.Container, addedCapabilities []corev1.Capability) {
	t.Helper()
	securityContext := container.SecurityContext
	if securityContext == nil || securityContext.AllowPrivilegeEscalation == nil ||
		*securityContext.AllowPrivilegeEscalation || securityContext.ReadOnlyRootFilesystem == nil ||
		!*securityContext.ReadOnlyRootFilesystem || securityContext.RunAsNonRoot == nil ||
		!*securityContext.RunAsNonRoot {
		t.Fatalf("container %q does not have the required non-root security context: %#v", container.Name, securityContext)
	}
	if securityContext.Capabilities == nil ||
		!reflect.DeepEqual(securityContext.Capabilities.Drop, []corev1.Capability{"ALL"}) ||
		!reflect.DeepEqual(securityContext.Capabilities.Add, addedCapabilities) {
		t.Fatalf("container %q has unexpected capabilities: %#v", container.Name, securityContext.Capabilities)
	}
}

func assertRunAsGroup(t *testing.T, container corev1.Container, expected int64) {
	t.Helper()
	if container.SecurityContext == nil || container.SecurityContext.RunAsGroup == nil ||
		*container.SecurityContext.RunAsGroup != expected {
		t.Fatalf("container %q does not run with GID %d: %#v", container.Name, expected, container.SecurityContext)
	}
}

func assertRunAsUser(t *testing.T, container corev1.Container, expected int64) {
	t.Helper()
	if container.SecurityContext == nil || container.SecurityContext.RunAsUser == nil ||
		*container.SecurityContext.RunAsUser != expected {
		t.Fatalf("container %q does not run with UID %d: %#v", container.Name, expected, container.SecurityContext)
	}
}

func assertGraylogDataPermissions(t *testing.T, statefulSet *appsv1.StatefulSet, openShift bool) {
	t.Helper()
	setup := statefulSet.Spec.Template.Spec.InitContainers[0]
	setupCommand := strings.Join(setup.Command, "\n")
	assertCommandContains(t, setupCommand, "chown -R 1001:1001 /data/db", "chmod -R u=rwX,g=rwX,o= /data/db")
	if !hasWritableGraylogMount(setup.VolumeMounts, "mongodb", "/data/db") {
		t.Fatalf("setup container does not mount the MongoDB data volume read-write: %#v", setup.VolumeMounts)
	}
	if openShift {
		assertCommandContains(t, setupCommand, "chmod -R 0777", "chmod 0666")
		return
	}
	assertCommandExcludes(t, setupCommand, "chmod -R 0777", "chmod 0666")
	assertCommandContains(t, setupCommand, "chown -R 1100:1100", "chmod -R u=rwX,g=rwX,o=", "chmod 0660")
}

func assertCommandContains(t *testing.T, command string, expectedValues ...string) {
	t.Helper()
	for _, expected := range expectedValues {
		if !strings.Contains(command, expected) {
			t.Fatalf("setup command does not contain %q: %s", expected, command)
		}
	}
}

func assertCommandExcludes(t *testing.T, command string, forbiddenValues ...string) {
	t.Helper()
	for _, forbidden := range forbiddenValues {
		if strings.Contains(command, forbidden) {
			t.Fatalf("setup command contains forbidden mode %q: %s", forbidden, command)
		}
	}
}

func hasWritableGraylogMount(mounts []corev1.VolumeMount, name, path string) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path && !mount.ReadOnly {
			return true
		}
	}
	return false
}

func assertBoundedEmptyDirs(t *testing.T, volumes []corev1.Volume) {
	t.Helper()
	for _, volume := range volumes {
		if volume.EmptyDir != nil && volume.EmptyDir.SizeLimit == nil {
			t.Fatalf("emptyDir volume %q has no size limit", volume.Name)
		}
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
