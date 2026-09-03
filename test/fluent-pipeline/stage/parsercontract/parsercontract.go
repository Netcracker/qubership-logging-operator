package parsercontract

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Manifest struct {
	Cases []Case `json:"cases"`
}

type Case struct {
	ID       string                 `json:"id"`
	Parser   string                 `json:"parser"`
	Match    bool                   `json:"match"`
	Input    string                 `json:"input"`
	Expected map[string]interface{} `json:"expected"`
	Absent   []string               `json:"absent,omitempty"`
}

var safeID = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func Prepare(manifestPath, parsersPath, targetDir string) error {
	manifest, err := ReadManifest(manifestPath)
	if err != nil {
		return err
	}
	for _, name := range []string{"input", "expected", "output"} {
		directory := filepath.Join(targetDir, name)
		if err := os.MkdirAll(directory, 0o777); err != nil {
			return err
		}
		// The helper and logging-agent containers use different UIDs. These directories contain disposable test artifacts.
		if err := os.Chmod(directory, 0o777); err != nil {
			return err
		}
	}

	parsers, err := os.ReadFile(parsersPath)
	if err != nil {
		return fmt.Errorf("read rendered parsers: %w", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "parsers.conf"), parsers, 0o644); err != nil {
		return err
	}
	parserNames, err := ParserNames(parsersPath)
	if err != nil {
		return err
	}
	available := make(map[string]bool, len(parserNames))
	for _, parser := range parserNames {
		available[parser] = true
	}

	var config strings.Builder
	config.WriteString("[SERVICE]\n    Flush 1\n    Log_Level error\n    Parsers_File /fluent-bit/etc/parsers.conf\n\n")
	expectations := make([]map[string]interface{}, 0, len(manifest.Cases))
	for _, testCase := range manifest.Cases {
		if err := validateCase(testCase); err != nil {
			return err
		}
		if !available[testCase.Parser] {
			continue
		}
		inputName := testCase.ID + ".log"
		if err := os.WriteFile(filepath.Join(targetDir, "input", inputName), []byte(testCase.Input+"\n"), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(&config, "[INPUT]\n    Name tail\n    Tag contract.%s\n    Path /parser-input/%s\n    Parser %s\n    Read_from_Head On\n    Refresh_Interval 1\n\n", testCase.ID, inputName, testCase.Parser)
		fmt.Fprintf(&config, "[FILTER]\n    Name record_modifier\n    Match contract.%s\n    Record test_case %s\n\n", testCase.ID, testCase.ID)

		expected := make(map[string]interface{}, len(testCase.Expected)+1)
		for key, value := range testCase.Expected {
			expected[key] = value
		}
		expected["_test"] = map[string]interface{}{
			"id":     testCase.ID,
			"match":  map[string]interface{}{"test_case": testCase.ID},
			"absent": testCase.Absent,
		}
		expectations = append(expectations, expected)
	}
	config.WriteString("[OUTPUT]\n    Name file\n    Match contract.*\n    Format plain\n    Path /parser-output\n    File output-log\n")

	if err := os.WriteFile(filepath.Join(targetDir, "fluent-bit.conf"), []byte(config.String()), 0o644); err != nil {
		return err
	}
	expectedJSON, err := json.MarshalIndent(expectations, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(targetDir, "expected", "parser-contracts.log.json"), append(expectedJSON, '\n'), 0o644)
}

func ParserNames(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var parsers []string
	inParser := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			inParser = line == "[PARSER]"
			continue
		}
		fields := strings.Fields(line)
		if inParser && len(fields) == 2 && fields[0] == "Name" && fields[1] != "logId-test" {
			parsers = append(parsers, fields[1])
		}
	}
	return parsers, scanner.Err()
}

func ReadManifest(path string) (Manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse parser contract manifest: %w", err)
	}
	return manifest, nil
}

func validateCase(testCase Case) error {
	if !safeID.MatchString(testCase.ID) {
		return fmt.Errorf("parser contract ID %q must match %s", testCase.ID, safeID)
	}
	if testCase.Parser == "" || strings.ContainsAny(testCase.Parser, " \t\r\n") {
		return fmt.Errorf("parser contract %q has invalid parser %q", testCase.ID, testCase.Parser)
	}
	if testCase.Input == "" || strings.ContainsAny(testCase.Input, "\r\n") {
		return fmt.Errorf("parser contract %q input must contain exactly one non-empty line", testCase.ID)
	}
	return nil
}

func CoveredParsers(manifest Manifest) map[string]map[bool]bool {
	covered := make(map[string]map[bool]bool)
	for _, testCase := range manifest.Cases {
		if covered[testCase.Parser] == nil {
			covered[testCase.Parser] = make(map[bool]bool)
		}
		covered[testCase.Parser][testCase.Match] = true
	}
	return covered
}

func MissingCases(manifest Manifest, parsers []string) []string {
	covered := CoveredParsers(manifest)
	var missing []string
	for _, parser := range parsers {
		if !covered[parser][true] {
			missing = append(missing, parser+": matching")
		}
		if !covered[parser][false] {
			missing = append(missing, parser+": non-matching")
		}
	}
	sort.Strings(missing)
	return missing
}
