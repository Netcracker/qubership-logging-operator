package fluentbit_forwarder_aggregator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestAggregatorOutputConfigSecretRendering(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(*loggingService.OutputFluentbit)
		mountPath  string
		volumeName string
		secretName string
		secretKey  string
		configName string
		include    string
	}{
		{
			name: "loki",
			configure: func(output *loggingService.OutputFluentbit) {
				output.Loki = &loggingService.LokiFluentbit{
					Enabled: true,
					ConfigSecret: &loggingService.OutputConfigSecret{
						SecretName: "aggregator-loki-output",
						SecretKey:  "output-loki.conf",
					},
				}
			},
			mountPath:  "/fluent-bit/secret-outputs/loki",
			volumeName: "loki-output-config",
			secretName: "aggregator-loki-output",
			secretKey:  "output-loki.conf",
			configName: "output-loki.conf",
			include:    "@INCLUDE /fluent-bit/secret-outputs/loki/output-loki.conf",
		},
		{
			name: "http",
			configure: func(output *loggingService.OutputFluentbit) {
				output.Http = &loggingService.HttpFluentbit{
					Enabled: true,
					Routing: &loggingService.FluentbitHTTPRouting{},
					ConfigSecret: &loggingService.OutputConfigSecret{
						SecretName: "aggregator-http-output",
						SecretKey:  "output-http.conf",
					},
				}
			},
			mountPath:  "/fluent-bit/secret-outputs/http",
			volumeName: "http-output-config",
			secretName: "aggregator-http-output",
			secretKey:  "output-http.conf",
			configName: "output-http.conf",
			include:    "@INCLUDE /fluent-bit/secret-outputs/http/output-http.conf",
		},
		{
			name: "otel",
			configure: func(output *loggingService.OutputFluentbit) {
				output.Otel = &loggingService.OtelFluentbit{
					Enabled: true,
					ConfigSecret: &loggingService.OutputConfigSecret{
						SecretName: "aggregator-otel-output",
						SecretKey:  "output-opentelemetry.conf",
					},
				}
			},
			mountPath:  "/fluent-bit/secret-outputs/otel",
			volumeName: "otel-output-config",
			secretName: "aggregator-otel-output",
			secretKey:  "output-opentelemetry.conf",
			configName: "output-opentelemetry.conf",
			include:    "@INCLUDE /fluent-bit/secret-outputs/otel/output-opentelemetry.conf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := newAggregatorRenderTestLoggingService()
			tt.configure(cr.Spec.Fluentbit.Aggregator.Output)

			ss, cm := renderAggregator(t, cr)

			assertAggregatorVolumeMount(t, ss, "logging-fluentbit-aggregator", tt.mountPath, tt.volumeName)
			assertAggregatorSecretVolume(t, ss, tt.volumeName, tt.secretName, tt.secretKey, tt.secretKey)
			assertAggregatorConfigContains(t, cm, "fluent-bit.conf", tt.include)
			assertAggregatorConfigAbsent(t, cm, tt.configName)
		})
	}
}

func TestAggregatorDefaultOutputConfigRendering(t *testing.T) {
	cr := newAggregatorRenderTestLoggingService()
	cr.Spec.Fluentbit.Aggregator.Output.Loki = &loggingService.LokiFluentbit{
		Enabled: true,
	}

	_, cm := renderAggregator(t, cr)

	assertAggregatorConfigContains(t, cm, "fluent-bit.conf", "@INCLUDE /fluent-bit/etc/output-loki.conf")
	assertAggregatorConfigPresent(t, cm, "output-loki.conf")
}

func TestAggregatorRenderedResources(t *testing.T) {
	outputDir := os.Getenv("FLUENTBIT_AGGREGATOR_RENDER_OUTPUT_DIR")
	if outputDir == "" {
		t.Skip("FLUENTBIT_AGGREGATOR_RENDER_OUTPUT_DIR is not set")
	}

	cr := newAggregatorRenderTestLoggingService()
	cr.Spec.Fluentbit.Aggregator.Output.Loki = &loggingService.LokiFluentbit{
		Enabled: true,
		ConfigSecret: &loggingService.OutputConfigSecret{
			SecretName: "aggregator-loki-output",
			SecretKey:  "output-loki.conf",
		},
	}

	ss, cm := renderAggregator(t, cr)
	writeAggregatorRenderedResource(t, outputDir, "fluentbit-aggregator-statefulset.yaml", ss)
	writeAggregatorRenderedResource(t, outputDir, "fluentbit-aggregator-configmap.yaml", cm)
}

func newAggregatorRenderTestLoggingService() *loggingService.LoggingService {
	return &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging", Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				DockerImage: "fluent-bit:test",
				Aggregator: &loggingService.FluentbitAggregator{
					DockerImage:     "fluent-bit:test",
					ConfigmapReload: &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
					Output:          &loggingService.OutputFluentbit{},
					Resources: &corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
			},
		},
	}
}

func renderAggregator(t *testing.T, cr *loggingService.LoggingService) (*appsv1.StatefulSet, *corev1.ConfigMap) {
	t.Helper()

	ss, err := aggregatorStatefulSet(cr)
	if err != nil {
		t.Fatal(err)
	}
	cm, err := aggregatorConfigMap(cr, util.DynamicParameters{})
	if err != nil {
		t.Fatal(err)
	}

	return ss, cm
}

func assertAggregatorConfigContains(t *testing.T, cm *corev1.ConfigMap, name, want string) {
	t.Helper()
	if !strings.Contains(cm.Data[name], want) {
		t.Fatalf("missing %q in %s:\n%s", want, name, cm.Data[name])
	}
}

func assertAggregatorConfigAbsent(t *testing.T, cm *corev1.ConfigMap, name string) {
	t.Helper()
	if _, ok := cm.Data[name]; ok {
		t.Fatalf("unexpected config %s", name)
	}
}

func assertAggregatorConfigPresent(t *testing.T, cm *corev1.ConfigMap, name string) {
	t.Helper()
	if _, ok := cm.Data[name]; !ok {
		t.Fatalf("missing config %s", name)
	}
}

func assertAggregatorVolumeMount(t *testing.T, ss *appsv1.StatefulSet, containerName, mountPath, volumeName string) {
	t.Helper()
	for _, container := range ss.Spec.Template.Spec.Containers {
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

func assertAggregatorSecretVolume(t *testing.T, ss *appsv1.StatefulSet, volumeName, secretName, key, path string) {
	t.Helper()
	for _, volume := range ss.Spec.Template.Spec.Volumes {
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

func writeAggregatorRenderedResource(t *testing.T, outputDir, name string, value any) {
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
