package kubeapi

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAnswer(t *testing.T) {
	t.Parallel()

	metadataDir := t.TempDir()
	annotated := `{"kind":"Pod","metadata":{"name":"annotated","namespace":"demo",` +
		`"annotations":{"fluentbit.io/parser":"logfmt"}}}`
	if err := os.WriteFile(filepath.Join(metadataDir, "demo_annotated.json"), []byte(annotated), 0o600); err != nil {
		t.Fatalf("write the metadata file: %v", err)
	}

	tests := []struct {
		name           string
		path           string
		wantStatus     int
		wantAnnotation interface{}
	}{
		{
			name:           "a pod with a metadata file is answered with its annotations",
			path:           "/api/v1/namespaces/demo/pods/annotated",
			wantStatus:     http.StatusOK,
			wantAnnotation: "logfmt",
		},
		{
			name:           "a pod with no metadata file is answered without annotations",
			path:           "/api/v1/namespaces/demo/pods/plain",
			wantStatus:     http.StatusOK,
			wantAnnotation: nil,
		},
		{
			name:       "a request that is not a pod request is refused",
			path:       "/api/v1/nodes",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertAnswer(t, metadataDir, tt.path, tt.wantStatus, tt.wantAnnotation)
		})
	}
}

func assertAnswer(t *testing.T, metadataDir, path string, wantStatus int, wantAnnotation interface{}) {
	t.Helper()

	recorder := httptest.NewRecorder()
	answer(metadataDir, recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != wantStatus {
		t.Fatalf("answer(%q) status = %d, want %d", path, recorder.Code, wantStatus)
	}
	if wantStatus != http.StatusOK {
		return
	}

	var pod struct {
		Metadata struct {
			Name        string            `json:"name"`
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &pod); err != nil {
		t.Fatalf("answer(%q) returned a body that is not a pod: %v", path, err)
	}
	got, exists := pod.Metadata.Annotations["fluentbit.io/parser"]
	if wantAnnotation == nil {
		if exists {
			t.Errorf("answer(%q) annotation = %q, want none", path, got)
		}
		return
	}
	if got != wantAnnotation {
		t.Errorf("answer(%q) annotation = %q, want %v", path, got, wantAnnotation)
	}
}

func TestWriteCredentialsProducesACertificateTheAgentTrusts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	certificate, err := writeCredentials(dir)
	if err != nil {
		t.Fatalf("writeCredentials() returned error: %v", err)
	}
	if len(certificate.Certificate) == 0 {
		t.Fatal("writeCredentials() returned no certificate")
	}

	authority := x509.NewCertPool()
	pemBytes, err := os.ReadFile(filepath.Join(dir, "ca.crt"))
	if err != nil {
		t.Fatalf("read the written CA: %v", err)
	}
	if !authority.AppendCertsFromPEM(pemBytes) {
		t.Fatal("the written ca.crt holds no certificate")
	}

	parsed, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		t.Fatalf("parse the returned certificate: %v", err)
	}
	if _, err := parsed.Verify(x509.VerifyOptions{DNSName: certificateHost, Roots: authority}); err != nil {
		t.Errorf("the certificate does not verify for %q against the written CA: %v", certificateHost, err)
	}
	if _, err := tls.X509KeyPair(pemBytes, mustRead(t, filepath.Join(dir, "tls.key"))); err != nil {
		t.Errorf("the written certificate and key are not a pair: %v", err)
	}
	if token := mustRead(t, filepath.Join(dir, "token")); len(token) == 0 {
		t.Error("writeCredentials() wrote an empty token")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}
