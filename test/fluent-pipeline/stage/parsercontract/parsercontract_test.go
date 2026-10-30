package parsercontract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestCoversEveryFluentBitParser(t *testing.T) {
	t.Parallel()

	manifest := checkedInManifest(t)
	for _, configuration := range shippedConfigurations(t) {
		if missing := MissingCases(manifest, configuration.parsers); len(missing) > 0 {
			t.Errorf("%s: parser contract cases are missing: %s", configuration.name, strings.Join(missing, ", "))
		}
	}
}

func TestManifestHasNoCaseForARemovedParser(t *testing.T) {
	t.Parallel()

	manifest := checkedInManifest(t)
	var shipped []string
	for _, configuration := range shippedConfigurations(t) {
		shipped = append(shipped, configuration.parsers...)
	}
	if stale := StaleCases(manifest, shipped); len(stale) > 0 {
		t.Fatalf("parser contract cases name a parser no parsers.conf defines: %s", strings.Join(stale, ", "))
	}
}

func TestStaleCases(t *testing.T) {
	t.Parallel()

	manifest := Manifest{Cases: []Case{
		{ID: "json-matching", Parser: "json", Match: true},
		{ID: "java-matching", Parser: "java", Match: true},
		{ID: "java-non-matching", Parser: "java", Match: false},
	}}

	got := StaleCases(manifest, []string{"json", "syslog"})
	want := []string{"java-matching (java)", "java-non-matching (java)"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("StaleCases() = %v, want %v", got, want)
	}
}

func checkedInManifest(t *testing.T) Manifest {
	t.Helper()

	manifest, err := ReadManifest(filepath.Join("..", "..", "testdata", "parser-cases.json"))
	if err != nil {
		t.Fatalf("read parser contract manifest: %v", err)
	}
	return manifest
}

type shippedConfiguration struct {
	name    string
	parsers []string
}

// shippedConfigurations returns the parsers of every Fluent Bit configuration the operator ships.
// A case runs against a configuration only where that configuration defines its parser, and the
// three define different sets and, for several names, different expressions. Coverage is therefore
// counted per configuration rather than over the union of their names.
func shippedConfigurations(t *testing.T) []shippedConfiguration {
	t.Helper()

	controllers := filepath.Join("..", "..", "..", "..", "controllers")
	sources := []struct {
		name string
		path string
	}{
		{"daemon set", filepath.Join(controllers, "fluentbit", "fluentbit.configmap", "conf.d", "parsers.conf")},
		{"forwarder", filepath.Join(controllers, "fluentbit-forwarder-aggregator", "forwarder.configmap", "conf.d", "parsers.conf")},
		{"aggregator", filepath.Join(controllers, "fluentbit-forwarder-aggregator", "aggregator.configmap", "conf.d", "parsers.conf")},
	}
	configurations := make([]shippedConfiguration, 0, len(sources))
	for _, source := range sources {
		parsers, err := ParserNames(source.path)
		if err != nil {
			t.Fatalf("read Fluent Bit parsers from %s: %v", source.path, err)
		}
		configurations = append(configurations, shippedConfiguration{name: source.name, parsers: parsers})
	}
	return configurations
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
