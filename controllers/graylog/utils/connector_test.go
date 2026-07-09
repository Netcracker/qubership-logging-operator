package utils

import (
	"context"
	"embed"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"k8s.io/client-go/kubernetes/fake"
)

//go:embed config/archives.json
var testAssets embed.FS

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

func TestReadSecretFileValueReturnsErrorForMissingFile(t *testing.T) {
	if _, err := readSecretFileValue(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("readSecretFileValue() returned nil error for missing file")
	}
}

func TestReadOperatorGraylogCredsReturnsErrorForMissingPasswordFile(t *testing.T) {
	secretDir := t.TempDir()
	setOperatorGraylogSecretFiles(t, secretDir)
	writeSecretFile(t, filepath.Join(secretDir, "user"), "admin")

	if _, err := readOperatorGraylogCreds(); err == nil {
		t.Fatal("readOperatorGraylogCreds() returned nil error for missing password file")
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

func TestManageArchivesDirectoryUsesOpenSearchHostFromSpec(t *testing.T) {
	body := manageArchivesDirectoryRequestBody(t, &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Graylog: &loggingService.Graylog{
				OpenSearch: &loggingService.OpenSearch{
					Host: "http://opensearch:9200",
				},
			},
		},
	})

	if !strings.Contains(body, `"type": "fs"`) {
		t.Fatalf("archive request body does not contain repository type: %s", body)
	}
	if !strings.Contains(body, "/usr/share/opensearch/snapshots/graylog") {
		t.Fatalf("archive request body was not rewritten for OpenSearch: %s", body)
	}
}

func TestManageArchivesDirectoryReadsOpenSearchHostFromSecretFile(t *testing.T) {
	secretDir := t.TempDir()
	setOperatorGraylogSecretFiles(t, secretDir)
	writeSecretFile(t, filepath.Join(secretDir, "elasticsearchHost"), "http://opensearch:9200")

	body := manageArchivesDirectoryRequestBody(t, &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Graylog: &loggingService.Graylog{},
		},
	})

	if !strings.Contains(body, "/usr/share/opensearch/snapshots/graylog") {
		t.Fatalf("archive request body was not rewritten for OpenSearch from secret host: %s", body)
	}
}

func TestManageArchivesDirectoryReturnsErrorForMissingSecretFile(t *testing.T) {
	secretDir := t.TempDir()
	setOperatorGraylogSecretFiles(t, secretDir)

	connector := &GraylogConnector{
		OpenSearchRestClient: &util.RestClient{
			Client: http.DefaultClient,
			Host:   "http://opensearch:9200",
		},
		Assets: testAssets,
		Log:    util.Logger("test-connector"),
	}

	err := connector.ManageArchivesDirectory(&loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Graylog: &loggingService.Graylog{},
		},
	})
	if err == nil {
		t.Fatal("ManageArchivesDirectory() returned nil error for missing secret file")
	}
}

func manageArchivesDirectoryRequestBody(t *testing.T, cr *loggingService.LoggingService) string {
	t.Helper()

	var requestBody string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPut {
			t.Fatalf("request method = %s, want %s", r.Method, http.MethodPut)
		}
		if r.URL.Path != "/_snapshot/archives" {
			t.Fatalf("request path = %s, want /_snapshot/archives", r.URL.Path)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		requestBody = string(data)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})

	connector := &GraylogConnector{
		OpenSearchRestClient: &util.RestClient{
			Client: &http.Client{Transport: transport},
			Host:   "http://opensearch:9200",
		},
		Assets: testAssets,
		Log:    util.Logger("test-connector"),
	}

	if err := connector.ManageArchivesDirectory(cr); err != nil {
		t.Fatalf("ManageArchivesDirectory() returned error: %v", err)
	}
	return requestBody
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
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
