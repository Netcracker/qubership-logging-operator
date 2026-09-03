package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

const testMetadataKey = "_test"

type testMetadata struct {
	ID     string                 `json:"id"`
	Match  map[string]interface{} `json:"match"`
	Absent []string               `json:"absent"`
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

func findActualRecord(actualRecords []map[string]interface{}, expected map[string]interface{}) (
	map[string]interface{}, bool, bool,
) {
	if metadata, ok := getTestMetadata(expected); ok && len(metadata.Match) > 0 {
		record, duplicated := singleRecord(matchingRecords(actualRecords, func(actual map[string]interface{}) bool {
			return isSubset(metadata.Match, actual)
		}))
		return record, duplicated, true
	}

	logID := expected["logId"]
	if logID == nil {
		logID = expected["_test_id"]
	}
	if logID != nil {
		matches := matchingRecords(actualRecords, func(actual map[string]interface{}) bool {
			return actual["logId"] == logID
		})
		matches = refineByTimestamp(matches, expected)
		if record, duplicated := singleRecord(matches); record != nil || duplicated {
			return record, duplicated, false
		}
	}

	if logID != nil {
		markers := [][]byte{
			[]byte("[logId=" + fmt.Sprint(logID) + "]"),
			[]byte("logId=" + fmt.Sprint(logID)),
		}
		matches := matchingRecords(actualRecords, func(actual map[string]interface{}) bool {
			encoded, err := json.Marshal(actual)
			if err != nil {
				return false
			}
			for _, marker := range markers {
				if bytes.Contains(encoded, marker) {
					return true
				}
			}
			return false
		})
		matches = refineByTimestamp(matches, expected)
		if record, duplicated := singleRecord(matches); record != nil || duplicated {
			return record, duplicated, true
		}
	}

	// Some parsers intentionally consume the suffix containing the injected test marker.
	// Input fixtures have stable timestamps, so a unique time is a safe final selector.
	record, duplicated := singleRecord(matchingRecords(actualRecords, func(actual map[string]interface{}) bool {
		return expected["time"] != nil && actual["time"] == expected["time"]
	}))
	return record, duplicated, record != nil
}

func compareRecord(expected, actual map[string]interface{}) (bool, error) {
	metadata, partial := getTestMetadata(expected)
	if !partial {
		return reflect.DeepEqual(expected, actual), nil
	}

	expectedFields := make(map[string]interface{}, len(expected)-1)
	for key, value := range expected {
		if key != testMetadataKey {
			expectedFields[key] = value
		}
	}
	if !isSubset(expectedFields, actual) {
		return false, nil
	}
	for _, field := range metadata.Absent {
		if _, exists := lookupField(actual, field); exists {
			return false, nil
		}
	}
	return true, nil
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

func refineByTimestamp(records []map[string]interface{}, expected map[string]interface{}) []map[string]interface{} {
	if len(records) < 2 {
		return records
	}
	for _, field := range []string{"time", "log_time", "fluentd_time", "date"} {
		if expected[field] == nil {
			continue
		}
		refined := matchingRecords(records, func(actual map[string]interface{}) bool {
			return actual[field] == expected[field]
		})
		if len(refined) > 0 {
			return refined
		}
	}
	return records
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

func GetModificationFuncs(agent string, ignoreFluentdTimeFiles string) (rmFuncs []RecordModifyFunc) {
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

func printJsonRecord(logId string, record map[string]interface{}, expected bool) error {
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
		fmt.Printf("\u001B[32m--- Expected log. LogId=%s ---\u001B[0m", logId)
	} else {
		fmt.Printf("\u001B[33;20m--- Actual log. LogId=%s ---\u001B[0m", logId)
	}
	fmt.Println()
	fmt.Println(buf.String())
	return nil
}
