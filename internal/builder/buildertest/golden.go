// Package buildertest runs the golden-file tests of the component builders.
//
// Each LoggingService in internal/builder/testdata is one test case. A builder package
// builds its objects for every case and compares them with the YAML files in its own
// testdata/<case>/ directory. Run the tests with -args -update to rewrite those files
// after an intended change, and review the diff before committing it.
package buildertest

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/yaml"
)

var update = flag.Bool("update", false, "rewrite the expected builder output in testdata")

// casesDir is relative to a builder package directory such as internal/builder/graylog.
const casesDir = "../testdata"

// Build returns the objects a builder creates for cr, keyed by the expected file name
// without the extension. It returns nil when the builder has nothing to build for cr.
type Build func(cr *loggingService.LoggingService) map[string]any

// Run compares the output of build with the expected files for every test case.
func Run(t *testing.T, build Build) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(casesDir, "*.yaml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no test cases in %s: %v", casesDir, err)
	}
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".yaml")
		t.Run(name, func(t *testing.T) {
			cr := readCR(t, file)
			objects := build(cr)
			if len(objects) == 0 {
				t.Skip("the custom resource does not configure this component")
			}
			keys := make([]string, 0, len(objects))
			for key := range objects {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				compare(t, filepath.Join("testdata", name, key+".yaml"), objects[key])
			}
		})
	}
}

func readCR(t *testing.T, file string) *loggingService.LoggingService {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	cr := &loggingService.LoggingService{}
	if err = yaml.UnmarshalStrict(raw, cr); err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	return cr
}

func compare(t *testing.T, path string, object any) {
	t.Helper()
	got, err := yaml.Marshal(object)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if *update {
		if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create %s: %v", filepath.Dir(path), err)
		}
		if err = os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v; run the tests with -args -update to create it", path, err)
	}
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("%s differs from the builder output (-want +got):\n%s", path, diff)
	}
}
