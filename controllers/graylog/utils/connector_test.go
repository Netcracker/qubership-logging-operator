package utils

import (
	"context"
	"embed"
	"os"
	"path/filepath"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestReadSecretFileValueTrimsWhitespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte(" value \n"), 0600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	value, err := readSecretFileValue(path)
	if err != nil {
		t.Fatalf("readSecretFileValue() returned error: %v", err)
	}
	if value != "value" {
		t.Fatalf("readSecretFileValue() = %q, want %q", value, "value")
	}
}

func TestCreateConnectorReadsMountedSecretFiles(t *testing.T) {
	secretDir := t.TempDir()
	setOperatorGraylogSecretFiles(t, secretDir)

	writeSecretFile(t, filepath.Join(secretDir, "user"), "admin\n")
	writeSecretFile(t, filepath.Join(secretDir, "password"), "password\n")
	writeSecretFile(t, filepath.Join(secretDir, "elasticsearchHost"), "https://elastic:secret@opensearch:9200/logs?pretty=true\n")

	cr := &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Graylog: &loggingService.Graylog{
				TLS: &loggingService.GraylogTLS{
					HTTP: &loggingService.HTTPGraylogTLS{},
				},
			},
		},
	}
	cr.SetNamespace("logging")

	connector, err := CreateConnector(context.Background(), cr, embed.FS{}, fake.NewSimpleClientset())
	if err != nil {
		t.Fatalf("CreateConnector() returned error: %v", err)
	}

	if connector.RestClient.Auth.Name != "admin" {
		t.Fatalf("Graylog username = %q, want %q", connector.RestClient.Auth.Name, "admin")
	}
	if connector.RestClient.Auth.Password != "password" {
		t.Fatalf("Graylog password = %q, want %q", connector.RestClient.Auth.Password, "password")
	}
	if connector.OpenSearchRestClient.Auth.Name != "elastic" {
		t.Fatalf("OpenSearch username = %q, want %q", connector.OpenSearchRestClient.Auth.Name, "elastic")
	}
	if connector.OpenSearchRestClient.Auth.Password != "secret" {
		t.Fatalf("OpenSearch password = %q, want %q", connector.OpenSearchRestClient.Auth.Password, "secret")
	}
	if connector.OpenSearchRestClient.Host != "https://opensearch:9200/logs?pretty=true" {
		t.Fatalf("OpenSearch host = %q", connector.OpenSearchRestClient.Host)
	}
}

func setOperatorGraylogSecretFiles(t *testing.T, secretDir string) {
	t.Helper()

	oldDir := operatorGraylogSecretDir
	oldUserFile := operatorGraylogUserFile
	oldPasswordFile := operatorGraylogPasswordFile
	oldElasticHostFile := operatorElasticHostFile

	operatorGraylogSecretDir = secretDir
	operatorGraylogUserFile = filepath.Join(secretDir, "user")
	operatorGraylogPasswordFile = filepath.Join(secretDir, "password")
	operatorElasticHostFile = filepath.Join(secretDir, "elasticsearchHost")

	t.Cleanup(func() {
		operatorGraylogSecretDir = oldDir
		operatorGraylogUserFile = oldUserFile
		operatorGraylogPasswordFile = oldPasswordFile
		operatorElasticHostFile = oldElasticHostFile
	})
}

func writeSecretFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		t.Fatalf("write secret file %s: %v", path, err)
	}
}
