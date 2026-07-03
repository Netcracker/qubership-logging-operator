package fluentbit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestFluentbitOutputConfigSecretRendering(t *testing.T) {
	cr := newRenderTestLoggingService()
	cr.Spec.Fluentbit.Output.Http = &loggingService.HttpFluentbit{
		Enabled: true,
		Routing: &loggingService.FluentbitHTTPRouting{},
		ConfigSecret: &loggingService.OutputConfigSecret{
			SecretName: "fluentbit-http-output",
			SecretKey:  "output-http.conf",
		},
	}

	ds, cm := renderFluentbit(t, cr)

	assertFluentbitVolumeMount(t, ds, "logging-fluentbit", "/fluent-bit/secret-outputs/http", "http-output-config")
	assertFluentbitSecretVolume(t, ds, "http-output-config", "fluentbit-http-output", "output-http.conf", "output-http.conf")
	assertFluentbitConfigContains(t, cm, "fluent-bit.conf", "@INCLUDE /fluent-bit/secret-outputs/http/output-http.conf")
	assertFluentbitConfigAbsent(t, cm, "output-http.conf")
}

func TestFluentbitRenderedResources(t *testing.T) {
	outputDir := os.Getenv("FLUENTBIT_RENDER_OUTPUT_DIR")
	if outputDir == "" {
		t.Skip("FLUENTBIT_RENDER_OUTPUT_DIR is not set")
	}

	cr := newRenderTestLoggingService()
	cr.Spec.Fluentbit.Output.Http = &loggingService.HttpFluentbit{
		Enabled: true,
		Routing: &loggingService.FluentbitHTTPRouting{},
		ConfigSecret: &loggingService.OutputConfigSecret{
			SecretName: "fluentbit-http-output",
			SecretKey:  "output-http.conf",
		},
	}

	ds, cm := renderFluentbit(t, cr)
	writeFluentbitRenderedResource(t, outputDir, "fluentbit-daemonset.yaml", ds)
	writeFluentbitRenderedResource(t, outputDir, "fluentbit-configmap.yaml", cm)
}

func newRenderTestLoggingService() *loggingService.LoggingService {
	return &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging", Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				DockerImage:     "fluent-bit:test",
				ConfigmapReload: &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
				Output:          &loggingService.OutputFluentbit{},
			},
		},
	}
}

func renderFluentbit(t *testing.T, cr *loggingService.LoggingService) (*appsv1.DaemonSet, *corev1.ConfigMap) {
	t.Helper()

	ds, err := fluentbitDaemonSet(cr, util.DynamicParameters{})
	if err != nil {
		t.Fatal(err)
	}
	cm, err := fluentbitConfigMap(cr, util.DynamicParameters{})
	if err != nil {
		t.Fatal(err)
	}

	return ds, cm
}

func assertFluentbitConfigContains(t *testing.T, cm *corev1.ConfigMap, name, want string) {
	t.Helper()
	if !strings.Contains(cm.Data[name], want) {
		t.Fatalf("missing %q in %s:\n%s", want, name, cm.Data[name])
	}
}

func assertFluentbitConfigAbsent(t *testing.T, cm *corev1.ConfigMap, name string) {
	t.Helper()
	if _, ok := cm.Data[name]; ok {
		t.Fatalf("unexpected config %s", name)
	}
}

func assertFluentbitVolumeMount(t *testing.T, ds *appsv1.DaemonSet, containerName, mountPath, volumeName string) {
	t.Helper()
	for _, container := range ds.Spec.Template.Spec.Containers {
		if container.Name != containerName {
			continue
		}
		for _, mount := range container.VolumeMounts {
			if mount.MountPath == mountPath && mount.Name == volumeName && mount.SubPath == "" {
				return
			}
		}
	}
	t.Fatalf("missing mount path=%s volume=%s", mountPath, volumeName)
}

func assertFluentbitSecretVolume(t *testing.T, ds *appsv1.DaemonSet, volumeName, secretName, key, path string) {
	t.Helper()
	for _, volume := range ds.Spec.Template.Spec.Volumes {
		if volume.Name != volumeName || volume.Secret == nil || volume.Secret.SecretName != secretName {
			continue
		}
		for _, item := range volume.Secret.Items {
			if item.Key == key && item.Path == path {
				return
			}
		}
	}
	t.Fatalf("missing secret volume=%s secret=%s key=%s path=%s", volumeName, secretName, key, path)
}

func writeFluentbitRenderedResource(t *testing.T, outputDir, name string, value any) {
	t.Helper()

	data, err := yaml.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(outputDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(outputDir, name), data, 0600); err != nil {
		t.Fatal(err)
	}
}
