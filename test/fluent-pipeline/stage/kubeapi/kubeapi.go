// Package kubeapi serves the pod metadata that the Fluent Bit Kubernetes filter reads, so that a
// scenario can exercise the filter without a cluster. The filter reaches it by name: the runner
// points kubernetes.default.svc at this server and mounts the certificate it writes as the service
// account CA.
package kubeapi

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// podRequest matches the path the Kubernetes filter requests for one pod.
var podRequest = regexp.MustCompile(`^/api/v1/namespaces/([^/]+)/pods/([^/]+)$`)

const (
	certificateHost = "kubernetes.default.svc"
	serviceAccount  = "fake-service-account-token"
)

// Serve writes the credentials the agent needs into credentialsDir and answers pod requests from
// metadataDir. A request for a pod with no file there is answered with metadata that carries no
// annotations and no labels, which is what a pod that declares neither looks like.
func Serve(metadataDir, credentialsDir, address string) error {
	certificate, err := writeCredentials(credentialsDir)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              address,
		Handler:           http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { answer(metadataDir, w, r) }),
		TLSConfig:         &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12},
		ReadHeaderTimeout: 10 * time.Second,
	}
	slog.Info("Serving pod metadata", "address", address, "metadataDir", metadataDir)
	return server.ListenAndServeTLS("", "")
}

func answer(metadataDir string, w http.ResponseWriter, r *http.Request) {
	match := podRequest.FindStringSubmatch(r.URL.Path)
	if match == nil {
		slog.Warn("Unexpected request", "path", r.URL.Path)
		http.NotFound(w, r)
		return
	}
	namespace, pod := match[1], match[2]

	body, err := os.ReadFile(filepath.Join(metadataDir, namespace+"_"+pod+".json"))
	if err != nil {
		slog.Debug("No metadata file for the pod, answering without annotations", "namespace", namespace, "pod", pod)
		body, err = json.Marshal(map[string]interface{}{
			"kind":       "Pod",
			"apiVersion": "v1",
			"metadata":   map[string]interface{}{"name": pod, "namespace": namespace},
			"status":     map[string]interface{}{"phase": "Running"},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	slog.Info("Answered a pod request", "namespace", namespace, "pod", pod, "bytes", len(body))
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil {
		slog.Error("Could not write the answer", "err", err)
	}
}

// writeCredentials creates a self-signed certificate for the API server name and leaves it, its
// key, and a token in credentialsDir. The runner mounts that directory as the service account
// directory of the agent, so the agent trusts the server and has a token to send.
func writeCredentials(dir string) (tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: certificateHost},
		DNSNames:              []string{certificateHost},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return tls.Certificate{}, err
	}
	for name, content := range map[string][]byte{
		"ca.crt":  certificatePEM,
		"tls.crt": certificatePEM,
		"tls.key": keyPEM,
		"token":   []byte(serviceAccount + "\n"),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
			return tls.Certificate{}, fmt.Errorf("write %s: %w", name, err)
		}
	}
	return tls.X509KeyPair(certificatePEM, keyPEM)
}
