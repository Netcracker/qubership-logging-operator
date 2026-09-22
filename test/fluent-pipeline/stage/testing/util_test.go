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
	t.Parallel()

	actual := []map[string]interface{}{
		{"time": "2024-01-01T00:00:00Z", "short_message": "first"},
		{"time": "2024-01-01T00:00:01Z", "short_message": "second"},
		{"log": "syslog line", "tag": "/var/log/syslog", "short_message": "third"},
		{"log": "syslog line", "tag": "/var/log/messages", "short_message": "fourth"},
	}

	tests := []struct {
		name          string
		expected      map[string]interface{}
		wantMessage   interface{}
		wantDuplicate bool
	}{
		{
			name:        "the default match field selects by the runtime timestamp",
			expected:    map[string]interface{}{"_test": metadata("one"), "time": "2024-01-01T00:00:01Z"},
			wantMessage: "second",
		},
		{
			name:        "a declared match field replaces the default",
			expected:    map[string]interface{}{"_test": metadata("two", "tag"), "tag": "/var/log/messages"},
			wantMessage: "fourth",
		},
		{
			name:        "several declared match fields must all hold",
			expected:    map[string]interface{}{"_test": metadata("three", "log", "tag"), "log": "syslog line", "tag": "/var/log/syslog"},
			wantMessage: "third",
		},
		{
			name:        "a value no output record carries selects nothing",
			expected:    map[string]interface{}{"_test": metadata("four"), "time": "2024-01-01T00:00:09Z"},
			wantMessage: nil,
		},
		{
			name:          "a value two output records carry is reported as duplicated",
			expected:      map[string]interface{}{"_test": metadata("five", "log"), "log": "syslog line"},
			wantMessage:   "first",
			wantDuplicate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testMetadata, ok := getTestMetadata(tt.expected)
			if !ok {
				t.Fatal("getTestMetadata() found no metadata in the expected record")
			}
			values, missing := selectorValues(tt.expected, testMetadata)
			if missing != "" {
				t.Fatalf("selectorValues() reported %q as missing, want no missing field", missing)
			}

			record, duplicated := findActualRecord(actual, values)
			if duplicated != tt.wantDuplicate {
				t.Errorf("findActualRecord(%v) duplicated = %v, want %v", values, duplicated, tt.wantDuplicate)
			}
			if tt.wantMessage == nil {
				if record != nil {
					t.Errorf("findActualRecord(%v) = %v, want no record", values, record)
				}
				return
			}
			if record == nil {
				t.Fatalf("findActualRecord(%v) = no record, want short_message %q", values, tt.wantMessage)
			}
			if got := record["short_message"]; got != tt.wantMessage && !tt.wantDuplicate {
				t.Errorf("findActualRecord(%v) short_message = %v, want %v", values, got, tt.wantMessage)
			}
		})
	}
}

func TestSelectorValuesReportsAMatchFieldTheExpectedRecordDoesNotDefine(t *testing.T) {
	t.Parallel()

	expected := map[string]interface{}{"_test": metadata("one", "log_time"), "time": "2024-01-01T00:00:00Z"}
	testMetadata, _ := getTestMetadata(expected)

	values, missing := selectorValues(expected, testMetadata)
	if missing != "log_time" {
		t.Errorf("selectorValues() missing = %q, want %q", missing, "log_time")
	}
	if values != nil {
		t.Errorf("selectorValues() = %v, want no values", values)
	}
}

func TestCompareRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected map[string]interface{}
		actual   map[string]interface{}
		want     bool
	}{
		{
			name:     "a whole-record expectation accepts the same fields",
			expected: map[string]interface{}{"_test": metadata("one"), "time": "one", "parse_format": "json"},
			actual:   map[string]interface{}{"time": "one", "parse_format": "json"},
			want:     true,
		},
		{
			name:     "a whole-record expectation rejects an extra output field",
			expected: map[string]interface{}{"_test": metadata("two"), "time": "one", "parse_format": "json"},
			actual:   map[string]interface{}{"time": "one", "parse_format": "json", "hostname": "generated"},
			want:     false,
		},
		{
			name:     "a partial expectation accepts an extra output field",
			expected: map[string]interface{}{"_test": partialMetadata("three"), "parse_format": "json"},
			actual:   map[string]interface{}{"time": "one", "parse_format": "json", "hostname": "generated"},
			want:     true,
		},
		{
			name:     "a partial expectation rejects a field it lists as absent",
			expected: map[string]interface{}{"_test": partialMetadata("four", "error"), "parse_format": "json"},
			actual:   map[string]interface{}{"parse_format": "json", "error": "unexpected"},
			want:     false,
		},
		{
			name:     "a partial expectation rejects a nested field it lists as absent",
			expected: map[string]interface{}{"_test": partialMetadata("five", "nested.secret"), "parse_format": "json"},
			actual:   map[string]interface{}{"parse_format": "json", "nested": map[string]interface{}{"secret": "leaked"}},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testMetadata, ok := getTestMetadata(tt.expected)
			if !ok {
				t.Fatal("getTestMetadata() found no metadata in the expected record")
			}
			if got := compareRecord(tt.expected, tt.actual, testMetadata); got != tt.want {
				t.Errorf("compareRecord(%v, %v) = %v, want %v", tt.expected, tt.actual, got, tt.want)
			}
		})
	}
}

// metadata builds the identification block of an expected record, with the match fields a case
// declares and the default ones when it declares none.
func metadata(id string, matchOn ...string) map[string]interface{} {
	block := map[string]interface{}{"id": id}
	if len(matchOn) > 0 {
		block["matchOn"] = toInterfaces(matchOn)
	}
	return block
}

// partialMetadata builds the identification block of a partial expectation, such as a generated
// parser contract, with the fields the output record must not carry.
func partialMetadata(id string, absent ...string) map[string]interface{} {
	block := metadata(id)
	block["partial"] = true
	if len(absent) > 0 {
		block["absent"] = toInterfaces(absent)
	}
	return block
}

func toInterfaces(values []string) []interface{} {
	encoded := make([]interface{}, 0, len(values))
	for _, value := range values {
		encoded = append(encoded, value)
	}
	return encoded
}

// TestTestJSONSuccess cannot run in parallel: testJson resolves paths relative to
// the working directory, so os.Chdir is required and must not race with other tests.
func TestTestJSONSuccess(t *testing.T) {
	dir := t.TempDir()
	writeFile(
		t,
		filepath.Join(dir, "output-logs", "actual", "output.log"),
		"{\"time\":\"one\",\"message\":\"ok\"}\nmetric_name = 1\n",
	)
	writeFile(t, filepath.Join(dir, "output-logs", "expected", "sample.log.json"), "[{\"_test\":{\"id\":\"one\"},\"time\":\"one\",\"message\":\"ok\"}]")

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

// TestTestJSONDuplicateSelectorFails cannot run in parallel: same reason as TestTestJSONSuccess.
func TestTestJSONDuplicateSelectorFails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "output-logs", "actual", "output.log"), "{\"time\":\"one\",\"message\":\"ok\"}\n{\"time\":\"one\",\"message\":\"ok\"}")
	writeFile(t, filepath.Join(dir, "output-logs", "expected", "sample.log.json"), "[{\"_test\":{\"id\":\"one\"},\"time\":\"one\",\"message\":\"ok\"}]")

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
