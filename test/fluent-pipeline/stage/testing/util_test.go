package testing

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
)

type stubAgent struct {
	outputFileName string
}

func (s stubAgent) UpdateCustomConfiguration(data map[string]string, _ *loggingService.LoggingService) map[string]string {
	return data
}

func (s stubAgent) GetOutputFileName() string {
	return s.outputFileName
}

func TestIgnoreFluentdTimeFunc(t *testing.T) {
	t.Parallel()

	modify := ignoreFluentdTimeFunc("audit.log.json")

	// File is in the ignore list: fluentd_time in expected should be overwritten with the actual value.
	expected := map[string]interface{}{"fluentd_time": "expected"}
	actual := map[string]interface{}{"fluentd_time": "actual"}
	if err := modify(expected, actual, "audit.log.json"); err != nil {
		t.Fatalf("ignoreFluentdTimeFunc returned error: %v", err)
	}
	if expected["fluentd_time"] != "actual" {
		t.Fatalf("fluentd_time = %v, want %v", expected["fluentd_time"], "actual")
	}

	// File is NOT in the ignore list: expected should remain unchanged.
	expected2 := map[string]interface{}{"fluentd_time": "expected"}
	actual2 := map[string]interface{}{"fluentd_time": "actual"}
	if err := modify(expected2, actual2, "other.log.json"); err != nil {
		t.Fatalf("ignoreFluentdTimeFunc returned error: %v", err)
	}
	if expected2["fluentd_time"] != "expected" {
		t.Fatalf("fluentd_time should not be changed for non-ignored file, got %v", expected2["fluentd_time"])
	}
}

func TestGetModificationFuncs(t *testing.T) {
	t.Parallel()

	if got := GetModificationFuncs("fluentd", "audit.log.json"); len(got) != 1 {
		t.Fatalf("GetModificationFuncs(fluentd) len = %d, want 1", len(got))
	}
	if got := GetModificationFuncs("fluentbit", "audit.log.json"); len(got) != 0 {
		t.Fatalf("GetModificationFuncs(fluentbit) len = %d, want 0", len(got))
	}
}

func TestApplyModificationFuncsStopsOnError(t *testing.T) {
	wantErr := errors.New("boom")
	calls := 0
	err := applyModificationFuncs(
		map[string]interface{}{},
		map[string]interface{}{},
		"file.log.json",
		[]RecordModifyFunc{
			func(expected, actual map[string]interface{}, file string) error {
				calls++
				return wantErr
			},
			func(expected, actual map[string]interface{}, file string) error {
				calls++
				return nil
			},
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("applyModificationFuncs error = %v, want %v", err, wantErr)
	}
	if calls != 1 {
		t.Fatalf("applyModificationFuncs calls = %d, want 1", calls)
	}
}

func TestContains(t *testing.T) {
	if !contains([]string{"a", "b"}, "b") {
		t.Fatal("contains returned false for existing element")
	}
	if contains([]string{"a", "b"}, "c") {
		t.Fatal("contains returned true for missing element")
	}
}

func TestFindActualRecord(t *testing.T) {
	tests := []struct {
		name          string
		actual        []map[string]interface{}
		expected      map[string]interface{}
		wantMessage   string
		wantTestOnly  bool
		wantDuplicate bool
	}{
		{
			name: "explicit selector",
			actual: []map[string]interface{}{
				{"time": "2024-01-01T00:00:00Z", "short_message": "selected"},
			},
			expected: map[string]interface{}{
				"_test": map[string]interface{}{
					"id":    "parser-positive",
					"match": map[string]interface{}{"time": "2024-01-01T00:00:00Z"},
				},
			},
			wantMessage:  "selected",
			wantTestOnly: true,
		},
		{
			name: "log ID field",
			actual: []map[string]interface{}{
				{"logId": "one", "short_message": "first"},
			},
			expected:    map[string]interface{}{"logId": "one"},
			wantMessage: "first",
		},
		{
			name: "embedded marker",
			actual: []map[string]interface{}{
				{"short_message": "message [logId=two]"},
			},
			expected:     map[string]interface{}{"logId": "two"},
			wantMessage:  "message [logId=two]",
			wantTestOnly: true,
		},
		{
			name: "test ID in an unparsed key-value message",
			actual: []map[string]interface{}{
				{"message": "level=info logId=raw-one msg=test", "short_message": "raw"},
			},
			expected:     map[string]interface{}{"_test_id": "raw-one"},
			wantMessage:  "raw",
			wantTestOnly: true,
		},
		{
			name: "stable time",
			actual: []map[string]interface{}{
				{"time": "2024-01-01T00:00:00Z", "short_message": "third"},
			},
			expected: map[string]interface{}{
				"logId": "three",
				"time":  "2024-01-01T00:00:00Z",
			},
			wantMessage:  "third",
			wantTestOnly: true,
		},
		{
			name: "duplicate log ID refined by time",
			actual: []map[string]interface{}{
				{"logId": "timed", "time": "one", "short_message": "first"},
				{"logId": "timed", "time": "two", "short_message": "second"},
			},
			expected: map[string]interface{}{
				"logId": "timed",
				"time":  "two",
			},
			wantMessage: "second",
		},
		{
			name: "duplicate log ID refined by parsed log time",
			actual: []map[string]interface{}{
				{"logId": "timed", "log_time": "one", "short_message": "first"},
				{"logId": "timed", "log_time": "two", "short_message": "second"},
			},
			expected: map[string]interface{}{
				"logId":    "timed",
				"log_time": "two",
			},
			wantMessage: "second",
		},
		{
			name: "duplicate",
			actual: []map[string]interface{}{
				{"logId": "four", "short_message": "first"},
				{"logId": "four", "short_message": "second"},
			},
			expected:      map[string]interface{}{"logId": "four"},
			wantMessage:   "first",
			wantDuplicate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, duplicated, testOnly := findActualRecord(tt.actual, tt.expected)
			if record == nil {
				t.Fatal("findActualRecord returned no record")
			}
			if got := record["short_message"]; got != tt.wantMessage {
				t.Fatalf("short_message = %v, want %q", got, tt.wantMessage)
			}
			if duplicated != tt.wantDuplicate {
				t.Fatalf("duplicated = %v, want %v", duplicated, tt.wantDuplicate)
			}
			if testOnly != tt.wantTestOnly {
				t.Fatalf("testOnly = %v, want %v", testOnly, tt.wantTestOnly)
			}
		})
	}
}

func TestCompareRecordPartial(t *testing.T) {
	t.Parallel()

	expected := map[string]interface{}{
		"_test": map[string]interface{}{
			"id":     "json-positive",
			"match":  map[string]interface{}{"time": "one"},
			"absent": []interface{}{"error", "nested.secret"},
		},
		"parse_format": "json",
		"nested": map[string]interface{}{
			"value": "kept",
		},
	}
	actual := map[string]interface{}{
		"time":         "one",
		"parse_format": "json",
		"nested": map[string]interface{}{
			"value": "kept",
			"extra": true,
		},
		"hostname": "generated",
	}

	equal, err := compareRecord(expected, actual)
	if err != nil {
		t.Fatalf("compareRecord returned error: %v", err)
	}
	if !equal {
		t.Fatal("compareRecord rejected a matching partial record")
	}

	actual["error"] = "unexpected"
	equal, err = compareRecord(expected, actual)
	if err != nil {
		t.Fatalf("compareRecord returned error: %v", err)
	}
	if equal {
		t.Fatal("compareRecord accepted a field listed as absent")
	}
}

func TestCompareRecordLegacyIsExact(t *testing.T) {
	t.Parallel()

	equal, err := compareRecord(
		map[string]interface{}{"logId": "one"},
		map[string]interface{}{"logId": "one", "extra": true},
	)
	if err != nil {
		t.Fatalf("compareRecord returned error: %v", err)
	}
	if equal {
		t.Fatal("compareRecord accepted extra fields for a legacy expectation")
	}
}

// TestTestJSONSuccess cannot run in parallel: testJson resolves paths relative to
// the working directory, so os.Chdir is required and must not race with other tests.
func TestTestJSONSuccess(t *testing.T) {
	dir := t.TempDir()
	writeFile(
		t,
		filepath.Join(dir, "output-logs", "actual", "output.log"),
		"{\"logId\":\"1\",\"message\":\"ok\"}\nmetric_name = 1\n",
	)
	writeFile(t, filepath.Join(dir, "output-logs", "expected", "sample.log.json"), "[{\"logId\":\"1\",\"message\":\"ok\"}]")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	success, err := testJson("", stubAgent{outputFileName: "output.log"}, nil)
	if err != nil {
		t.Fatalf("testJson returned error: %v", err)
	}
	if !success {
		t.Fatal("testJson success = false, want true")
	}
}

// TestTestJSONDuplicateLogIDFails cannot run in parallel: same reason as TestTestJSONSuccess.
func TestTestJSONDuplicateLogIDFails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "output-logs", "actual", "output.log"), "{\"logId\":\"1\",\"message\":\"ok\"}\n{\"logId\":\"1\",\"message\":\"ok\"}")
	writeFile(t, filepath.Join(dir, "output-logs", "expected", "sample.log.json"), "[{\"logId\":\"1\",\"message\":\"ok\"}]")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	success, err := testJson("", stubAgent{outputFileName: "output.log"}, nil)
	if err != nil {
		t.Fatalf("testJson returned error: %v", err)
	}
	if success {
		t.Fatal("testJson success = true, want false")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
