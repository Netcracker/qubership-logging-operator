package parsercontract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestCoversEveryFluentBitParser(t *testing.T) {
	t.Parallel()

	manifest, err := ReadManifest(filepath.Join("..", "..", "testdata", "parser-cases.json"))
	if err != nil {
		t.Fatalf("read parser contract manifest: %v", err)
	}
	var parsers []string
	for _, path := range []string{
		filepath.Join("..", "..", "..", "..", "controllers", "fluentbit", "fluentbit.configmap", "conf.d", "parsers.conf"),
		filepath.Join("..", "..", "..", "..", "controllers", "fluentbit-forwarder-aggregator", "forwarder.configmap", "conf.d", "parsers.conf"),
		filepath.Join("..", "..", "..", "..", "controllers", "fluentbit-forwarder-aggregator", "aggregator.configmap", "conf.d", "parsers.conf"),
	} {
		configured, err := ParserNames(path)
		if err != nil {
			t.Fatalf("read Fluent Bit parsers from %s: %v", path, err)
		}
		parsers = append(parsers, configured...)
	}

	if missing := MissingCases(manifest, parsers); len(missing) > 0 {
		t.Fatalf("parser contract cases are missing: %s", strings.Join(missing, ", "))
	}
}

func TestPrepare(t *testing.T) {
	t.Parallel()

	target := t.TempDir()
	manifestPath := filepath.Join(target, "cases.json")
	parsersPath := filepath.Join(target, "source-parsers.conf")
	manifest := `{"cases":[{"id":"json-matching","parser":"json","match":true,"input":"{\"message\":\"ok\"}","expected":{"message":"ok"},"absent":["log"]}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parsersPath, []byte("[PARSER]\n    Name json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	generated := filepath.Join(target, "generated")
	if err := Prepare(manifestPath, parsersPath, generated); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	config, err := os.ReadFile(filepath.Join(generated, "fluent-bit.conf"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"Parser json", "Record test_case json-matching", "File output-log"} {
		if !strings.Contains(string(config), fragment) {
			t.Errorf("generated config does not contain %q", fragment)
		}
	}
}
