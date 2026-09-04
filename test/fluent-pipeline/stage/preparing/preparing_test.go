package preparing

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Netcracker/qubership-logging-operator/test/fluent-pipeline/agent"
)

// loggingServiceCR is the minimal custom resource the tests render configuration from.
// YAML indents with spaces, so the editorconfig indentation check does not apply here.
//
// editorconfig-checker-disable
const loggingServiceCR = `
apiVersion: logging.netcracker.com/v1
kind: LoggingService
metadata:
  name: logging-service
spec:
  fluentbit:
    customInputConf: input
    customFilterConf: filter
    customOutputConf: output
`

// editorconfig-checker-enable

func TestReadCustomResource(t *testing.T) {
	t.Parallel()

	crPath := writeLoggingServiceCR(t, loggingServiceCR)
	cr, err := readCustomResource(crPath)
	if err != nil {
		t.Fatalf("readCustomResource returned error: %v", err)
	}
	if cr == nil {
		t.Fatal("readCustomResource returned nil CR")
	}
}

func TestReadCustomResourceInvalidYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	crPath := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(crPath, []byte("not: [valid"), 0o600); err != nil {
		t.Fatalf("write invalid CR: %v", err)
	}

	cr, err := readCustomResource(crPath)
	if err == nil {
		t.Fatal("readCustomResource error = nil, want error")
	}
	if cr != nil {
		t.Fatalf("readCustomResource returned %#v, want nil", cr)
	}
}

func TestSaveDataToDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	data := map[string]string{
		"a.conf": "first",
		"b.conf": "second",
	}

	if err := saveDataToDirectory(dir, data); err != nil {
		t.Fatalf("saveDataToDirectory returned error: %v", err)
	}

	for fileName, want := range data {
		got, err := os.ReadFile(filepath.Join(dir, fileName))
		if err != nil {
			t.Fatalf("read %s: %v", fileName, err)
		}
		if string(got) != want {
			t.Fatalf("%s content = %q, want %q", fileName, string(got), want)
		}
	}
}

func TestMakeGeneratedLogsWritable(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "pods")
	nested := filepath.Join(root, "namespace_pod_uid", "container")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatalf("create generated log directory: %v", err)
	}
	logPath := filepath.Join(nested, "0.log")
	if err := os.WriteFile(logPath, []byte("log"), 0o600); err != nil {
		t.Fatalf("write generated log: %v", err)
	}

	if err := makeGeneratedLogsWritable(root); err != nil {
		t.Fatalf("makeGeneratedLogsWritable returned error: %v", err)
	}

	dirInfo, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("stat generated log directory: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o777 {
		t.Fatalf("generated log directory permissions = %o, want 777", got)
	}
	logInfo, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("stat generated log: %v", err)
	}
	if got := logInfo.Mode().Perm(); got != 0o666 {
		t.Fatalf("generated log permissions = %o, want 666", got)
	}
}

func TestMakeGeneratedLogsWritableAllowsMissingDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if err := makeGeneratedLogsWritable(missing); err != nil {
		t.Fatalf("makeGeneratedLogsWritable(%q): %v", missing, err)
	}
}

func TestFillConfigurationTemplatesMissingDirectory(t *testing.T) {
	t.Parallel()

	_, err := fillConfigurationTemplates(filepath.Join(t.TempDir(), "missing"), nil)
	if err == nil {
		t.Fatal("fillConfigurationTemplates error = nil, want error")
	}
}

// TestGetConfigurationAddsAgentSpecificFiles cannot run in parallel because it
// modifies the package-level sourceConfigPath variable.
func TestGetConfigurationAddsAgentSpecificFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "base.conf"), []byte("static-content"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	originalSourceConfigPath := sourceConfigPath
	sourceConfigPath = dir
	t.Cleanup(func() { sourceConfigPath = originalSourceConfigPath })

	crPath := writeLoggingServiceCR(t, loggingServiceCR)
	cr, err := readCustomResource(crPath)
	if err != nil {
		t.Fatalf("readCustomResource returned error: %v", err)
	}

	data, err := getConfiguration(&agent.Fluentbit{}, cr)
	if err != nil {
		t.Fatalf("getConfiguration returned error: %v", err)
	}

	if data["base.conf"] != "static-content" {
		t.Fatalf("base.conf = %q, want %q", data["base.conf"], "static-content")
	}
	if data["filter-custom.conf"] == "" || data["output-custom.conf"] == "" {
		t.Fatalf("expected agent-specific configuration to be added, got %#v", data)
	}
}

func writeLoggingServiceCR(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "logging-service.yaml")
	if err := os.WriteFile(path, []byte(fmt.Sprintf("%s\n", content)), 0o600); err != nil {
		t.Fatalf("write logging service CR: %v", err)
	}
	return path
}
