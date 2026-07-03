package utils

import (
	"context"
	"embed"
	"fmt"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateConnectorUsesElasticsearchHostFromSpec(t *testing.T) {
	cr := newGraylogConnectorTestCR()
	cr.Spec.Graylog.ElasticsearchHost = fmt.Sprintf("https://admin:%s%s/search", "admin", "@opensearch.logging:9200")

	connector, err := CreateConnector(context.Background(), cr, embed.FS{}, fake.NewSimpleClientset())
	if err != nil {
		t.Fatalf("CreateConnector() error = %v", err)
	}

	if connector.OpenSearchRestClient.Host != "https://opensearch.logging:9200/search" {
		t.Fatalf("host = %q", connector.OpenSearchRestClient.Host)
	}
	if connector.OpenSearchRestClient.Auth.Name != "admin" {
		t.Fatalf("auth name = %q, want admin", connector.OpenSearchRestClient.Auth.Name)
	}
	if connector.OpenSearchRestClient.Auth.Password != "admin" {
		t.Fatalf("auth password = %q, want admin", connector.OpenSearchRestClient.Auth.Password)
	}
}

func TestCreateConnectorUsesOpenSearchHost(t *testing.T) {
	cr := newGraylogConnectorTestCR()
	cr.Spec.Graylog.OpenSearch = &loggingService.OpenSearch{
		Host: "http://opensearch.logging:9200",
	}

	connector, err := CreateConnector(context.Background(), cr, embed.FS{}, fake.NewSimpleClientset())
	if err != nil {
		t.Fatalf("CreateConnector() error = %v", err)
	}

	if connector.OpenSearchRestClient.Host != "http://opensearch.logging:9200" {
		t.Fatalf("host = %q", connector.OpenSearchRestClient.Host)
	}
	if connector.OpenSearchRestClient.Auth.Name != "admin" {
		t.Fatalf("auth name = %q, want admin", connector.OpenSearchRestClient.Auth.Name)
	}
}

func TestCreateConnectorRequiresGraylogCredentials(t *testing.T) {
	cr := newGraylogConnectorTestCR()
	cr.Spec.Graylog.User = ""

	if _, err := CreateConnector(context.Background(), cr, embed.FS{}, fake.NewSimpleClientset()); err == nil {
		t.Fatal("CreateConnector() error = nil, want error")
	}
}

func TestCreateConnectorRequiresOpenSearchHost(t *testing.T) {
	cr := newGraylogConnectorTestCR()

	if _, err := CreateConnector(context.Background(), cr, embed.FS{}, fake.NewSimpleClientset()); err == nil {
		t.Fatal("CreateConnector() error = nil, want error")
	}
}

func TestGetOpenSearchHost(t *testing.T) {
	cr := newGraylogConnectorTestCR()
	cr.Spec.Graylog.ElasticsearchHost = "http://legacy-opensearch:9200"
	cr.Spec.Graylog.OpenSearch = &loggingService.OpenSearch{
		Host: "http://opensearch:9200",
	}
	if got := getOpenSearchHost(cr); got != "http://legacy-opensearch:9200" {
		t.Fatalf("getOpenSearchHost() = %q", got)
	}

	cr.Spec.Graylog.ElasticsearchHost = ""
	if got := getOpenSearchHost(cr); got != "http://opensearch:9200" {
		t.Fatalf("getOpenSearchHost() = %q", got)
	}

	cr.Spec.Graylog.OpenSearch = nil
	if got := getOpenSearchHost(cr); got != "" {
		t.Fatalf("getOpenSearchHost() = %q", got)
	}
}

func newGraylogConnectorTestCR() *loggingService.LoggingService {
	return &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Graylog: &loggingService.Graylog{
				User:              "admin",
				Password:          "admin",
				GraylogSecretName: "graylog-secret",
				TLS: &loggingService.GraylogTLS{
					HTTP: &loggingService.HTTPGraylogTLS{},
				},
			},
		},
	}
}
