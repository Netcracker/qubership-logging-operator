package fluentd

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

func TestFluentdSecretFileAuthRendering(t *testing.T) {
	t.Run("http basic auth uses mounted files", func(t *testing.T) {
		cr := newRenderTestLoggingService()
		cr.Spec.Fluentd.Output.Http = &loggingService.HttpFluentd{
			Enabled: true,
			Host:    "http://victorialogs",
			Auth: &loggingService.Auth{
				User:     secretRef("http-auth", "username"),
				Password: secretRef("http-auth", "password"),
			},
		}

		ds, cm := renderFluentd(t, cr)

		assertEnvAbsent(t, ds, "HTTP_USERNAME", "HTTP_PASSWORD", "HTTP_TOKEN")
		assertVolumeMount(t, ds, "/fluentd/output/http/auth/username", "http-auth-user", "username")
		assertVolumeMount(t, ds, "/fluentd/output/http/auth/password", "http-auth-password", "password")
		assertSecretVolume(t, ds, "http-auth-user", "http-auth")
		assertSecretVolume(t, ds, "http-auth-password", "http-auth")
		assertContains(t, cm.Data["output-http.conf"], `username   "#{File.read('/fluentd/output/http/auth/username').strip}"`)
		assertContains(t, cm.Data["output-http.conf"], `password   "#{File.read('/fluentd/output/http/auth/password').strip}"`)
	})

	t.Run("http token auth uses mounted file", func(t *testing.T) {
		cr := newRenderTestLoggingService()
		cr.Spec.Fluentd.Output.Http = &loggingService.HttpFluentd{
			Enabled: true,
			Host:    "http://victorialogs",
			Auth: &loggingService.Auth{
				Token: secretRef("http-token", "token"),
			},
		}

		ds, cm := renderFluentd(t, cr)

		assertEnvAbsent(t, ds, "HTTP_USERNAME", "HTTP_PASSWORD", "HTTP_TOKEN")
		assertVolumeMount(t, ds, "/fluentd/output/http/auth/token", "http-auth-token", "token")
		assertSecretVolume(t, ds, "http-auth-token", "http-token")
		assertContains(t, cm.Data["output-http.conf"], `Authorization":"Bearer #{File.read('/fluentd/output/http/auth/token').strip}`)
	})

	t.Run("http basic auth has priority over token auth", func(t *testing.T) {
		cr := newRenderTestLoggingService()
		cr.Spec.Fluentd.Output.Http = &loggingService.HttpFluentd{
			Enabled: true,
			Host:    "http://victorialogs",
			Auth: &loggingService.Auth{
				User:     secretRef("http-auth", "username"),
				Password: secretRef("http-auth", "password"),
				Token:    secretRef("http-token", "token"),
			},
		}

		ds, cm := renderFluentd(t, cr)

		assertVolumeMount(t, ds, "/fluentd/output/http/auth/username", "http-auth-user", "username")
		assertVolumeMount(t, ds, "/fluentd/output/http/auth/password", "http-auth-password", "password")
		assertVolumeMountAbsent(t, ds, "/fluentd/output/http/auth/token")
		assertNotContains(t, cm.Data["output-http.conf"], "/fluentd/output/http/auth/token")
	})

	t.Run("loki basic and token auth use mounted files", func(t *testing.T) {
		cr := newRenderTestLoggingService()
		cr.Spec.Fluentd.Output.Loki = &loggingService.LokiFluentd{
			Enabled:       true,
			Host:          "http://loki",
			LabelsMapping: "namespace $.kubernetes.namespace_name",
			Auth: &loggingService.Auth{
				User:     secretRef("loki-auth", "username"),
				Password: secretRef("loki-auth", "password"),
				Token:    secretRef("loki-token", "token"),
			},
		}

		ds, cm := renderFluentd(t, cr)

		assertEnvAbsent(t, ds, "LOKI_USERNAME", "LOKI_PASSWORD", "LOKI_TOKEN")
		assertVolumeMount(t, ds, "/fluentd/output/loki/auth/username", "loki-auth-user", "username")
		assertVolumeMount(t, ds, "/fluentd/output/loki/auth/password", "loki-auth-password", "password")
		assertVolumeMount(t, ds, "/fluentd/output/loki/auth/token", "loki-auth-token", "token")
		assertSecretVolume(t, ds, "loki-auth-user", "loki-auth")
		assertSecretVolume(t, ds, "loki-auth-password", "loki-auth")
		assertSecretVolume(t, ds, "loki-auth-token", "loki-token")
		assertContains(t, cm.Data["output-loki.conf"], `bearer_token_file "/fluentd/output/loki/auth/token"`)
		assertContains(t, cm.Data["output-loki.conf"], `username "#{File.read('/fluentd/output/loki/auth/username').strip}"`)
		assertContains(t, cm.Data["output-loki.conf"], `password "#{File.read('/fluentd/output/loki/auth/password').strip}"`)
	})

	t.Run("writes rendered resources for workflow summary", func(t *testing.T) {
		outputDir := os.Getenv("FLUENTD_RENDER_OUTPUT_DIR")
		if outputDir == "" {
			t.Skip("FLUENTD_RENDER_OUTPUT_DIR is not set")
		}

		cr := newRenderTestLoggingService()
		cr.Spec.Fluentd.Output.Http = &loggingService.HttpFluentd{
			Enabled: true,
			Host:    "http://victorialogs",
			Auth: &loggingService.Auth{
				Token: secretRef("http-token", "token"),
			},
		}
		cr.Spec.Fluentd.Output.Loki = &loggingService.LokiFluentd{
			Enabled:       true,
			Host:          "http://loki",
			LabelsMapping: "namespace $.kubernetes.namespace_name",
			Auth: &loggingService.Auth{
				User:     secretRef("loki-auth", "username"),
				Password: secretRef("loki-auth", "password"),
				Token:    secretRef("loki-token", "token"),
			},
		}

		ds, cm := renderFluentd(t, cr)
		writeRenderedResource(t, outputDir, "fluentd-daemonset.yaml", ds)
		writeRenderedResource(t, outputDir, "fluentd-configmap.yaml", cm)
	})
}

func newRenderTestLoggingService() *loggingService.LoggingService {
	return &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging", Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			Fluentd: &loggingService.Fluentd{
				DockerImage:      "fluentd:test",
				ConfigmapReload:  &loggingService.ConfigmapReload{DockerImage: "configmap-reload:test"},
				Output:           &loggingService.OutputFluentd{},
				QueueLimitLength: 256,
			},
		},
	}
}

func renderFluentd(t *testing.T, cr *loggingService.LoggingService) (*appsv1.DaemonSet, *corev1.ConfigMap) {
	t.Helper()

	ds, err := fluentdDaemonSet(cr, util.DynamicParameters{})
	if err != nil {
		t.Fatal(err)
	}
	cm, err := fluentdConfigMap(cr, util.DynamicParameters{})
	if err != nil {
		t.Fatal(err)
	}

	return ds, cm
}

func secretRef(name, key string) *corev1.SecretKeySelector {
	return &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: name},
		Key:                  key,
	}
}

func writeRenderedResource(t *testing.T, outputDir, name string, value any) {
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

func assertContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("missing %q in:\n%s", want, text)
	}
}

func assertNotContains(t *testing.T, text, unwanted string) {
	t.Helper()
	if strings.Contains(text, unwanted) {
		t.Fatalf("unexpected %q in:\n%s", unwanted, text)
	}
}

func assertEnvAbsent(t *testing.T, ds *appsv1.DaemonSet, names ...string) {
	t.Helper()
	for _, container := range ds.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			for _, name := range names {
				if env.Name == name {
					t.Fatalf("unexpected env %s", name)
				}
			}
		}
	}
}

func assertVolumeMount(t *testing.T, ds *appsv1.DaemonSet, mountPath, volumeName, subPath string) {
	t.Helper()
	for _, container := range ds.Spec.Template.Spec.Containers {
		for _, mount := range container.VolumeMounts {
			if mount.MountPath == mountPath && mount.Name == volumeName && mount.SubPath == subPath {
				return
			}
		}
	}
	t.Fatalf("missing mount path=%s volume=%s subPath=%s", mountPath, volumeName, subPath)
}

func assertVolumeMountAbsent(t *testing.T, ds *appsv1.DaemonSet, mountPath string) {
	t.Helper()
	for _, container := range ds.Spec.Template.Spec.Containers {
		for _, mount := range container.VolumeMounts {
			if mount.MountPath == mountPath {
				t.Fatalf("unexpected mount path=%s", mountPath)
			}
		}
	}
}

func assertSecretVolume(t *testing.T, ds *appsv1.DaemonSet, volumeName, secretName string) {
	t.Helper()
	for _, volume := range ds.Spec.Template.Spec.Volumes {
		if volume.Name == volumeName && volume.Secret != nil && volume.Secret.SecretName == secretName {
			return
		}
	}
	t.Fatalf("missing volume=%s secret=%s", volumeName, secretName)
}
