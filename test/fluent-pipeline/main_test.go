package main

import (
	"log/slog"
	"reflect"
	"testing"

	"github.com/Netcracker/qubership-logging-operator/test/fluent-pipeline/agent"
)

func assertInitAgentResult(t *testing.T, input string, got any, ok bool, wantType any, wantValid bool) {
	t.Helper()

	if ok != wantValid {
		t.Fatalf("initAgent(%q) valid = %v, want %v", input, ok, wantValid)
	}
	if !wantValid {
		if got != nil {
			t.Fatalf("initAgent(%q) agent = %T, want nil", input, got)
		}
		return
	}
	if got == nil {
		t.Fatalf("initAgent(%q) agent is nil", input)
	}
	if reflect.TypeOf(got) != reflect.TypeOf(wantType) {
		t.Fatalf("initAgent(%q) type = %T, want %T", input, got, wantType)
	}
}

func TestInitAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantType  any
		wantValid bool
	}{
		{name: "fluentd", input: "fluentd", wantType: &agent.Fluentd{}, wantValid: true},
		{name: "fluentbit with dash", input: "fluent-bit", wantType: &agent.Fluentbit{}, wantValid: true},
		{name: "fluentbit ha mixed case", input: "FluentBitHA", wantType: &agent.FluentbitHA{}, wantValid: true},
		{name: "invalid", input: "vector", wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := initAgent(tt.input)
			assertInitAgentResult(t, tt.input, got, ok, tt.wantType, tt.wantValid)
		})
	}
}

func TestGetLogLevel(t *testing.T) {
	t.Parallel()

	if got := getLogLevel("debug"); got != slog.LevelDebug {
		t.Fatalf("getLogLevel(debug) = %v, want %v", got, slog.LevelDebug)
	}
	if got := getLogLevel("not-a-level"); got != slog.LevelInfo {
		t.Fatalf("getLogLevel(invalid) = %v, want %v", got, slog.LevelInfo)
	}
}

func TestReplaceAttrs(t *testing.T) {
	t.Parallel()

	sourceAttr := slog.Any(slog.SourceKey, &slog.Source{
		File: "/tmp/project/main.go",
		Line: 27,
	})
	got := replaceAttrs(nil, sourceAttr)
	if got.Key != slog.SourceKey {
		t.Fatalf("replaceAttrs source key = %q, want %q", got.Key, slog.SourceKey)
	}
	if got.Value.String() != "main.go:27" {
		t.Fatalf("replaceAttrs source value = %q, want %q", got.Value.String(), "main.go:27")
	}

	other := slog.String("component", "tests")
	if got := replaceAttrs(nil, other); got.Key != other.Key || got.Value.String() != other.Value.String() {
		t.Fatalf("replaceAttrs should keep non-source attrs unchanged, got %+v want %+v", got, other)
	}
}
