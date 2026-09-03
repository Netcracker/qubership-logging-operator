package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Netcracker/qubership-logging-operator/test/fluent-pipeline/agent"
	"github.com/Netcracker/qubership-logging-operator/test/fluent-pipeline/stage/parsercontract"
	"github.com/Netcracker/qubership-logging-operator/test/fluent-pipeline/stage/preparing"
	"github.com/Netcracker/qubership-logging-operator/test/fluent-pipeline/stage/testing"
)

var (
	agentFluentbit   = regexp.MustCompile(`^fluent-?bit$`)
	agentFluentbitHA = regexp.MustCompile(`^fluent-?bit-?ha$`)
)

func main() {
	stage := flag.String("stage", "test", "Stage of pipeline testing. Available values: prepare, test")
	agentString := flag.String("agent", "fluentbit", "Parse configuration of logging agent. Possible values: fluentbit, fluentbitha, fluentd")
	crPath := flag.String("cr", "/assets/logging-service-test-fluentbit.yaml", "Path to test LoggingService custom resource with necessary parameters")
	parserCasesPath := flag.String("parserCases", "/parser-contracts/cases.json", "Path to parser contract cases")
	parsersPath := flag.String("parsers", "/rendered-config/parsers.conf", "Path to the rendered Fluent Bit parser configuration")
	parserTargetPath := flag.String("parserTarget", "/parser-contracts/generated", "Directory for generated parser contract files")
	ignoreFiles := flag.String("ignore", "", "The list of files names that should be ignored during tests. The names must be separated with comma")
	ignoreFluentdTimeFiles := flag.String("ignoreFluentdTime", "audit.log.json,kubernetes.audit.log.json,varlogsyslog.log.json", "The list of files to ignore fluentd_time parameter because there is no way to check it for now")
	logLevel := flag.String("loglevel", "info", "Level of application logger")

	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: getLogLevel(*logLevel), ReplaceAttr: replaceAttrs, AddSource: true}))
	logger.With("app", "fluent-config-replacer")
	slog.SetDefault(logger)

	agent, ok := initAgent(*agentString)
	if !ok {
		logger.Error("Agent value is not valid", "agent", *agentString)
		os.Exit(1)
	}

	if strings.EqualFold(*stage, "test") {
		testing.CompareLogs(*ignoreFiles, agent, testing.GetModificationFuncs(*agentString, *ignoreFluentdTimeFiles))
	} else if strings.EqualFold(*stage, "prepare-parser-contracts") {
		if err := parsercontract.Prepare(*parserCasesPath, *parsersPath, *parserTargetPath); err != nil {
			logger.Error("Could not prepare parser contracts", "err", err)
			os.Exit(1)
		}
	} else if strings.EqualFold(*stage, "prepare") {
		preparing.PrepareConfiguration(*crPath, agent)
		preparing.PrepareTestLogs("/testdata/")
	} else {
		logger.Error("Stage of testing is not defined", "stage", *stage)
		os.Exit(1)
	}
}

func initAgent(agentString string) (agent.Agent, bool) {
	if strings.EqualFold(agentString, "fluentd") {
		return &agent.Fluentd{}, true
	} else if agentFluentbit.FindIndex([]byte(strings.ToLower(agentString))) != nil {
		return &agent.Fluentbit{}, true
	} else if agentFluentbitHA.FindIndex([]byte(strings.ToLower(agentString))) != nil {
		return &agent.FluentbitHA{}, true
	}
	return nil, false
}

func getLogLevel(logLevel string) slog.Level {
	var lvl slog.LevelVar
	if err := lvl.UnmarshalText([]byte(logLevel)); err != nil {
		return slog.LevelInfo
	}
	return lvl.Level()
}

func replaceAttrs(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.SourceKey {
		source := a.Value.Any().(*slog.Source)
		source.File = filepath.Base(source.File)
		return slog.Attr{
			Key:   slog.SourceKey,
			Value: slog.StringValue(fmt.Sprintf("%s:%v", filepath.Base(source.File), source.Line)),
		}
	}
	return a
}
