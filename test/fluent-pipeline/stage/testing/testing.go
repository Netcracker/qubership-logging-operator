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
	notDroppedStatus     = "\u001b[31mLog is in output but was expected to be dropped\u001B[0m"
)

// Directories where the runner mounts the records to compare, relative to the working directory
// of the fluent-pipeline-test container.
var (
	actualLogsDir   = filepath.Join("output-logs", "actual")
	expectedLogsDir = filepath.Join("output-logs", "expected")
)

type reportRow struct {
	id      string
	status  string
	details string
}

// comparison holds the state of a single run: the records read from the agent output, which of
// them an expected record has claimed, the modifications to apply before comparing, and the result
// collected for the final report.
type comparison struct {
	actualRecords      []map[string]interface{}
	claimed            []bool
	modificationFuncs  []RecordModifyFunc
	report             []reportRow
	filesWithoutTestID []string
	unexpected         []map[string]interface{}
	success            bool
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
	outputLogFileName := agent.GetOutputFileName()
	if len(outputLogFileName) == 0 {
		slog.Error("Could not get filename for agent", "agent", agent)
		return false, nil
	}

	slog.Info("Reading actual log file...", "filename", outputLogFileName)
	actualRecords, err := readActualRecords(filepath.Join(actualLogsDir, outputLogFileName))
	if err != nil {
		return false, err
	}

	run := &comparison{
		actualRecords:     actualRecords,
		claimed:           make([]bool, len(actualRecords)),
		modificationFuncs: modificationFuncs,
		success:           true,
	}

	slog.Info("Reading expected logs directory...", "dir", expectedLogsDir)
	if err := run.compareExpectedFiles(expectedLogsDir, strings.Split(ignore, ",")); err != nil {
		return false, err
	}
	run.collectUnexpected()

	slog.Info("Check finished!")
	run.printReport(agent)
	return run.success, nil
}

// readActualRecords reads the agent output line by line. The agent writes one JSON record per line
// and may add plain-text lines, such as metrics, that are not part of the processed logs.
func readActualRecords(path string) ([]map[string]interface{}, error) {
	actual, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var records []map[string]interface{}
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
			return nil, fmt.Errorf("parse pipeline output as JSON: %w", err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan pipeline output: %w", err)
	}
	return records, nil
}

func (c *comparison) compareExpectedFiles(root string, ignoreFiles []string) error {
	expectedFS := os.DirFS(root)
	return filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".log.json") {
			return nil
		}

		_, expectedFile := filepath.Split(path)
		expected, err := fs.ReadFile(expectedFS, expectedFile)
		if err != nil {
			c.success = false
			slog.Error("Error occurred while reading file", "path", path)
			return err
		}
		slog.Debug("Reading file", "path", path, "expectedFile", expectedFile)

		var expectedRecords []map[string]interface{}
		if err := json.Unmarshal(expected, &expectedRecords); err != nil {
			c.success = false
			return err
		}
		if contains(ignoreFiles, expectedFile) {
			slog.Info(fmt.Sprintf("Skipping file %s", expectedFile))
			c.claimRecords(expectedRecords)
			return nil
		}
		return c.compareRecords(expectedFile, expectedRecords)
	})
}

// claimRecords marks the output records of an ignored file as accounted for, so that skipping the
// comparison does not report them as unexpected.
func (c *comparison) claimRecords(expectedRecords []map[string]interface{}) {
	for _, record := range expectedRecords {
		metadata, ok := getTestMetadata(record)
		if !ok {
			continue
		}
		if values, missing := selectorValues(record, metadata); missing == "" {
			c.claim(findActualRecords(c.actualRecords, values))
		}
	}
}

func (c *comparison) claim(indexes []int) {
	for _, index := range indexes {
		c.claimed[index] = true
	}
}

// collectUnexpected gathers the output records no expected record claimed. Each fixture line the
// pipeline keeps has to be described, so an unclaimed record fails the run.
func (c *comparison) collectUnexpected() {
	for index, record := range c.actualRecords {
		if !c.claimed[index] {
			c.unexpected = append(c.unexpected, record)
		}
	}
	if len(c.unexpected) > 0 {
		c.success = false
	}
}

func (c *comparison) compareRecords(expectedFile string, expectedRecords []map[string]interface{}) error {
	for _, record := range expectedRecords {
		if err := c.compareExpectedRecord(expectedFile, record); err != nil {
			return err
		}
	}
	return nil
}

// compareExpectedRecord looks up the output record that belongs to one expected record and reports
// the outcome. It returns an error only when the comparison itself cannot continue; a mismatch is
// recorded in the report instead.
func (c *comparison) compareExpectedRecord(expectedFile string, record map[string]interface{}) error {
	metadata, ok := getTestMetadata(record)
	if !ok || metadata.ID == "" {
		c.success = false
		c.filesWithoutTestID = append(c.filesWithoutTestID, expectedFile)
		slog.Error(fmt.Sprintf("expected record in file %q has no %q metadata with an %q", expectedFile, testMetadataKey, "id"))
		return nil
	}

	values, missing := selectorValues(record, metadata)
	if missing != "" {
		c.fail(metadata.ID, fmt.Sprintf("expected record has no %q field to match on", missing))
		slog.Error(fmt.Sprintf("expected record %q in file %q has no %q field to match on", metadata.ID, expectedFile, missing))
		return nil
	}

	indexes := findActualRecords(c.actualRecords, values)
	c.claim(indexes)
	if metadata.Dropped {
		if len(indexes) == 0 {
			c.report = append(c.report, reportRow{id: metadata.ID, status: succeeded})
			return nil
		}
		c.fail(metadata.ID, notDroppedStatus)
		slog.Error(fmt.Sprintf("%d output record(s) match %v, but %q in file %q expects the pipeline to drop the line", len(indexes), values, metadata.ID, expectedFile))
		return nil
	}
	if len(indexes) > 1 {
		c.fail(metadata.ID, moreThanOneLogStatus)
		slog.Warn(fmt.Sprintf("Check of %q with id=%s failed: %v matched more than one output record", expectedFile, metadata.ID, values))
		return nil
	}
	if len(indexes) == 0 {
		c.fail(metadata.ID, logNotFoundStatus)
		slog.Error(fmt.Sprintf("no output record in file %q matches %v", expectedFile, values))
		return nil
	}
	actualRecord := c.actualRecords[indexes[0]]

	if err := applyModificationFuncs(record, actualRecord, expectedFile, c.modificationFuncs); err != nil {
		slog.Error("could not apply modification function to records")
	}

	if compareRecord(record, actualRecord, metadata) {
		c.report = append(c.report, reportRow{id: metadata.ID, status: succeeded})
		slog.Debug(fmt.Sprintf("Check logs from %q with id=%s is successful: record parsed", expectedFile, metadata.ID))
		return nil
	}

	c.fail(metadata.ID, differencesStatus)
	printMismatch(metadata.ID, expectedFile, recordDifferences(record, actualRecord, metadata))
	return nil
}

func (c *comparison) fail(id, details string) {
	c.success = false
	c.report = append(c.report, reportRow{id: id, status: failed, details: details})
}

// printMismatch reports the fields of one record that do not hold what the expected record asks
// for. The whole records are in the output file the run leaves behind; the report carries what
// moved, as expected -> produced.
func printMismatch(id, expectedFile string, differences []string) {
	fmt.Printf("\u001B[33;20m--- %s (%s): %d field(s) differ, expected -> produced ---\u001B[0m", id, expectedFile, len(differences))
	fmt.Println()
	for _, difference := range differences {
		fmt.Println("    " + difference)
	}
}

func (c *comparison) printReport(agent agent.Agent) {
	fmt.Printf("--- Report of %s pipeline testing ---", agent)
	fmt.Println()
	fmt.Println("ID\tSTATUS\tDETAILS")
	for _, row := range c.report {
		fmt.Printf("%v\t%s\t%s\n", row.id, row.status, row.details)
	}

	if len(c.filesWithoutTestID) > 0 {
		fmt.Printf("--- Files with an expected record that has no %q metadata ---", testMetadataKey)
		fmt.Println()
		for _, file := range c.filesWithoutTestID {
			fmt.Println(file)
		}
	}

	if len(c.unexpected) > 0 {
		fmt.Printf("--- %d output record(s) no expected record describes ---", len(c.unexpected))
		fmt.Println()
		for _, record := range c.unexpected {
			fmt.Println(describeRecord(record))
		}
	}
}
