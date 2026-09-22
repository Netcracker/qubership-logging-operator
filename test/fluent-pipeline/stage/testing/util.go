package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

const testMetadataKey = "_test"

// defaultMatchOn identifies a record by the timestamp that the container runtime wrote and the
// pipeline keeps. Every fixture record has a unique one, so it selects a single output record.
var defaultMatchOn = []string{"time"}

// testMetadata is the only way an expected record is identified. ID names the record in the
// report, MatchOn lists the expected fields whose values select the output record, and Partial
// compares the listed fields alone, which the generated parser contracts rely on.
type testMetadata struct {
	ID      string   `json:"id"`
	MatchOn []string `json:"matchOn"`
	Partial bool     `json:"partial"`
	Absent  []string `json:"absent"`
}

func getTestMetadata(expected map[string]interface{}) (testMetadata, bool) {
	raw, ok := expected[testMetadataKey]
	if !ok {
		return testMetadata{}, false
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return testMetadata{}, false
	}
	var metadata testMetadata
	if err := json.Unmarshal(encoded, &metadata); err != nil {
		return testMetadata{}, false
	}
	return metadata, true
}

func (m testMetadata) matchFields() []string {
	if len(m.MatchOn) == 0 {
		return defaultMatchOn
	}
	return m.MatchOn
}

// selectorValues returns the values that identify the record, read from the expected record
// itself. It also returns the first match field the expected record does not define, because such
// a record selects nothing.
func selectorValues(expected map[string]interface{}, metadata testMetadata) (map[string]interface{}, string) {
	fields := metadata.matchFields()
	values := make(map[string]interface{}, len(fields))
	for _, field := range fields {
		value, exists := lookupField(expected, field)
		if !exists {
			return nil, field
		}
		values[field] = value
	}
	return values, ""
}

// findActualRecord returns the output record that carries the selector values and whether more
// than one record carried them.
func findActualRecord(actualRecords []map[string]interface{}, values map[string]interface{}) (
	map[string]interface{}, bool,
) {
	return singleRecord(matchingRecords(actualRecords, func(actual map[string]interface{}) bool {
		for field, value := range values {
			actualValue, exists := lookupField(actual, field)
			if !exists || !reflect.DeepEqual(actualValue, value) {
				return false
			}
		}
		return true
	}))
}

func compareRecord(expected, actual map[string]interface{}, metadata testMetadata) bool {
	expectedFields := make(map[string]interface{}, len(expected))
	for key, value := range expected {
		if key != testMetadataKey {
			expectedFields[key] = value
		}
	}
	if !metadata.Partial {
		return reflect.DeepEqual(expectedFields, actual)
	}
	if !isSubset(expectedFields, actual) {
		return false
	}
	for _, field := range metadata.Absent {
		if _, exists := lookupField(actual, field); exists {
			return false
		}
	}
	return true
}

func isSubset(expected, actual map[string]interface{}) bool {
	for key, expectedValue := range expected {
		actualValue, ok := actual[key]
		if !ok {
			return false
		}
		expectedMap, nested := expectedValue.(map[string]interface{})
		if nested {
			actualMap, ok := actualValue.(map[string]interface{})
			if !ok || !isSubset(expectedMap, actualMap) {
				return false
			}
			continue
		}
		if !reflect.DeepEqual(expectedValue, actualValue) {
			return false
		}
	}
	return true
}

func lookupField(record map[string]interface{}, path string) (interface{}, bool) {
	var current interface{} = record
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func matchingRecords(actualRecords []map[string]interface{}, matches func(map[string]interface{}) bool) []map[string]interface{} {
	var found []map[string]interface{}
	for _, actual := range actualRecords {
		if matches(actual) {
			found = append(found, actual)
		}
	}
	return found
}

func singleRecord(records []map[string]interface{}) (map[string]interface{}, bool) {
	if len(records) == 0 {
		return nil, false
	}
	return records[0], len(records) > 1
}

type RecordModifyFunc func(expected, actual map[string]interface{}, file string) error

func ignoreFluentdTimeFunc(ignoreFluentdTimeFiles string) RecordModifyFunc {
	ignoreFiles := strings.Split(ignoreFluentdTimeFiles, ",")
	return func(expected, actual map[string]interface{}, file string) error {
		if contains(ignoreFiles, file) {
			expected["fluentd_time"] = actual["fluentd_time"]
		}
		return nil
	}
}

func GetModificationFuncs(agent, ignoreFluentdTimeFiles string) (rmFuncs []RecordModifyFunc) {
	if strings.EqualFold(agent, "fluentd") && len(ignoreFluentdTimeFiles) > 0 {
		rmFuncs = append(rmFuncs, ignoreFluentdTimeFunc(ignoreFluentdTimeFiles))
	}
	return
}

func applyModificationFuncs(record map[string]interface{}, actualRecord map[string]interface{}, file string, modificationFuncs []RecordModifyFunc) error {
	for _, applyFunc := range modificationFuncs {
		if err := applyFunc(record, actualRecord, file); err != nil {
			return err
		}
	}
	return nil
}

func contains(slc []string, el string) bool {
	for i := range slc {
		if el == slc[i] {
			return true
		}
	}
	return false
}

func printJsonRecord(id string, record map[string]interface{}, expected bool) error {
	src, err := json.Marshal(record)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = json.Indent(&buf, src, "", "\t")
	if err != nil {
		return err
	}
	if expected {
		fmt.Printf("\u001B[32m--- Expected log. id=%s ---\u001B[0m", id)
	} else {
		fmt.Printf("\u001B[33;20m--- Actual log. id=%s ---\u001B[0m", id)
	}
	fmt.Println()
	fmt.Println(buf.String())
	return nil
}
