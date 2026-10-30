package utils

import (
	"context"
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestUpdatePasswordSyncsRootPasswordHashWhenPasswordDidNotChange(t *testing.T) {
	clientset := fake.NewSimpleClientset(graylogSecret("admin"))
	watcher := &SecretEventWatcher{
		Clientset: clientset,
		Log:       util.Logger("test-watch-secret"),
	}
	cr := graylogWatchTestCR("admin")

	if err := watcher.updatePassword(graylogSecret("admin"), cr, nil); err != nil {
		t.Fatalf("updatePassword() error = %v", err)
	}

	secret, err := clientset.CoreV1().Secrets("logging").Get(context.Background(), "graylog-secret", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if got := string(secret.Data[util.GraylogSecretKeyRootPasswordSHA2]); got != adminSHA256 {
		t.Fatalf("rootPasswordSha2 = %q, want %q", got, adminSHA256)
	}
}

func TestUpdatePasswordUpdatesConfigMapAndRootPasswordHash(t *testing.T) {
	clientset := fake.NewSimpleClientset(graylogSecret("new-admin"), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GraylogComponentName,
			Namespace: "logging",
		},
		Data: map[string]string{
			util.GraylogConfigFileName: "root_password_sha2 = old\nroot_username = admin",
		},
	})
	watcher := &SecretEventWatcher{
		Clientset: clientset,
		Log:       util.Logger("test-watch-secret"),
	}
	cr := graylogWatchTestCR("admin")

	if err := watcher.updatePassword(graylogSecret("new-admin"), cr, nil); err != nil {
		t.Fatalf("updatePassword() error = %v", err)
	}
	if cr.Spec.Graylog.Password != "new-admin" {
		t.Fatalf("cr password = %q, want new-admin", cr.Spec.Graylog.Password)
	}

	configMap, err := clientset.CoreV1().ConfigMaps("logging").Get(context.Background(), util.GraylogComponentName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get configmap: %v", err)
	}
	if !strings.Contains(configMap.Data[util.GraylogConfigFileName], "root_password_sha2 = ") {
		t.Fatalf("graylog config was not updated: %q", configMap.Data[util.GraylogConfigFileName])
	}

	secret, err := clientset.CoreV1().Secrets("logging").Get(context.Background(), "graylog-secret", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if got := string(secret.Data[util.GraylogSecretKeyRootPasswordSHA2]); got == "" {
		t.Fatal("rootPasswordSha2 is empty")
	}
}

func graylogSecret(password string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "graylog-secret",
			Namespace: "logging",
		},
		Data: map[string][]byte{
			"password": []byte(password),
		},
	}
}

func graylogWatchTestCR(password string) *loggingService.LoggingService {
	return &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "logging-service",
			Namespace: "logging",
		},
		Spec: loggingService.LoggingServiceSpec{
			Graylog: &loggingService.Graylog{
				Password:          password,
				GraylogSecretName: "graylog-secret",
			},
		},
	}
}
