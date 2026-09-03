package testing

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Netcracker/qubership-logging-operator/test/fluent-pipeline/agent"
)

var (
	succeeded            = "\u001b[32mSucceeded\u001B[0m"
	failed               = "\u001b[31mFailed\u001B[0m"
	differencesStatus    = "\u001b[33;20mLog is processed, but with differences\u001B[0m"
	logNotFoundStatus    = "\u001b[31mLog is not found in output\u001B[0m"
	moreThanOneLogStatus = "\u001b[31mMore than one matching log in output\u001B[0m"
)

type reportRow struct {
	logID   interface{}
	status  string
	details string
}

func CompareLogs(ignoreFiles string, agent agent.Agent, modificationFuncs []RecordModifyFunc) {
	success, err := testJson(ignoreFiles, agent, modificationFuncs)
	if err != nil {
		slog.Error("Error occurred when reading output logs", "err", err)
		os.Exit(1)
	}
	if !success {
		slog.Error("Fluent pipeline test failed. See the preceding logs")
		os.Exit(1)
	}
	slog.Info("Check finished successfully!")
}

func testJson(ignore string, agent agent.Agent, modificationFuncs []RecordModifyFunc) (bool, error) {
	success := true
	ignoreFiles := strings.Split(ignore, ",")

	outputLogFileName := agent.GetOutputFileName()
	if len(outputLogFileName) == 0 {
		slog.Error("Could not get filename for agent", "agent", agent)
		return false, nil
	}

	slog.Info("Reading actual log file...", "filename", outputLogFileName)

	actual, err := os.ReadFile(filepath.Join("output-logs", "actual", outputLogFileName)) //fluent-pipeline-test/output-logs
	if err != nil {
		return false, err
	}
	var resultJsonStruct []map[string]interface{}
	scanner := bufio.NewScanner(bytes.NewReader(actual))
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if line[0] != '{' {
			slog.Debug("Ignoring non-JSON pipeline output", "content", string(line))
			continue
		}
		var record map[string]interface{}
		if err := json.Unmarshal(line, &record); err != nil {
			return false, fmt.Errorf("parse pipeline output as JSON: %w", err)
		}
		resultJsonStruct = append(resultJsonStruct, record)
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("scan pipeline output: %w", err)
	}
	pathExpectedLogs := filepath.Join("output-logs", "expected") //fluent-pipeline-test/output-logs
	outputLogsDir := os.DirFS(pathExpectedLogs)
	slog.Info("Reading expected logs directory...", "dir", pathExpectedLogs)

	var report []reportRow
	var filesWithoutLogID []string
	err = filepath.Walk(pathExpectedLogs, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".log.json") {
			_, expectedFile := filepath.Split(path)
			if contains(ignoreFiles, expectedFile) {
				slog.Info(fmt.Sprintf("Skipping file %s", expectedFile))
				return nil
			}
			expected, err := fs.ReadFile(outputLogsDir, expectedFile)
			if err != nil {
				success = false
				slog.Error("Error occurred while reading file", "path", path)
				return err
			}
			slog.Debug("Reading file", "path", path, "expectedFile", expectedFile)

			var expectedJsonStruct []map[string]interface{}
			err = json.Unmarshal(expected, &expectedJsonStruct)
			if err != nil {
				success = false
				return err
			}
			//compare actual and expected
			for _, record := range expectedJsonStruct {
				// Parse the test ID and use it to find the corresponding output record.
				logId := record["logId"]
				if metadata, ok := getTestMetadata(record); ok && metadata.ID != "" {
					logId = metadata.ID
				}
				if record["_test_id"] != nil {
					logId = record["_test_id"]
				}
				if logId != nil {
					actualRecord, isDuplicated, logIDIsTestOnly := findActualRecord(resultJsonStruct, record)
					if isDuplicated {
						success = false
						slog.Warn(fmt.Sprintf("Check of %q with logId=%s failed: more than one matching output record was found", expectedFile, logId))
						report = append(report, reportRow{logID: logId, status: failed, details: moreThanOneLogStatus})
					} else if !isDuplicated && actualRecord != nil {
						delete(record, "_test_id")
						if logIDIsTestOnly {
							delete(record, "logId")
						}
						if err := applyModificationFuncs(record, actualRecord, expectedFile, modificationFuncs); err != nil {
							slog.Error("could not apply modification function to records")
						}
						isEqual, compareErr := compareRecord(record, actualRecord)
						if compareErr != nil {
							return compareErr
						}
						if isEqual {
							//everything is ok!
							report = append(report, reportRow{logID: logId, status: succeeded})
							slog.Debug(fmt.Sprintf("Check logs from %q with logId=%s is successful: record parsed", expectedFile, logId))
						} else {
							//check failed
							success = false
							report = append(report, reportRow{logID: logId, status: failed, details: differencesStatus})
							slog.Warn(fmt.Sprintf("Check logs from %q with logId=%s is failed. Expected log printed below", expectedFile, logId))
							err = printJsonRecord(fmt.Sprintf("%v", logId), record, true)
							if err != nil {
								slog.Error("Error occurred while printing log record", "err", err)
								return err
							}
							slog.Warn("Actual log printed below")
							err = printJsonRecord(fmt.Sprintf("%v", logId), actualRecord, false)
							if err != nil {
								slog.Error("Error occurred while printing log record", "err", err)
								return err
							}
						}
						actualRecord = nil
						continue
					} else {
						//check failed
						success = false
						report = append(report, reportRow{logID: logId, status: failed, details: logNotFoundStatus})
						slog.Error(fmt.Sprintf("could not find logId with value %s in file %q with actual logs", logId, expectedFile))
					}
				} else {
					//check failed
					success = false
					filesWithoutLogID = append(filesWithoutLogID, expectedFile)
					slog.Error(fmt.Sprintf("could not find %q in file %q with expected logs", "logId", expectedFile))
				}
			}
		}
		return nil
	})
	if err != nil {
		success = false
		return success, err
	}
	slog.Info("Check finished!")
	fmt.Printf("--- Report of %s pipeline testing ---", agent)
	fmt.Println()
	fmt.Println("LOG ID\tSTATUS\tDETAILS")
	for _, row := range report {
		fmt.Printf("%v\t%s\t%s\n", row.logID, row.status, row.details)
	}

	if len(filesWithoutLogID) > 0 {
		fmt.Printf("--- Files where %q was not found ---", "logId")
		fmt.Println()
		for _, file := range filesWithoutLogID {
			fmt.Println(file)
		}
	}

	return success, err
}
