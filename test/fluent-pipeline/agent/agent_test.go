package agent

import (
	"reflect"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
)

func TestFluentbitUpdateCustomConfiguration(t *testing.T) {
	t.Parallel()

	cr := newFluentbitCR()
	data := map[string]string{"existing.conf": "keep"}

	got := (&Fluentbit{}).UpdateCustomConfiguration(data, cr)

	if got["existing.conf"] != "keep" {
		t.Fatalf("existing configuration entry was changed")
	}
	if got["input-custom.conf"] == "" || got["filter-custom.conf"] == "" || got["output-custom.conf"] == "" {
		t.Fatalf("expected fluentbit custom configuration to be populated, got %#v", got)
	}
}

func TestFluentdUpdateCustomConfiguration(t *testing.T) {
	t.Parallel()

	cr := newFluentdCR()
	got := (&Fluentd{}).UpdateCustomConfiguration(map[string]string{}, cr)

	if got["filter-custom.conf"] == "" || got["output-custom.conf"] == "" {
		t.Fatalf("expected fluentd custom configuration to be populated, got %#v", got)
	}
}

// TestHARolesRenderTheirOwnCustomConfiguration pins what each role of the HA deployment receives.
// The custom resource gives the two roles different values, so a renderer that reads the fields of
// the other role fails rather than passing on a shared value.
func TestHARolesRenderTheirOwnCustomConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		agent Agent
		want  map[string]string
	}{
		{
			name:  "the forwarder renders the custom sections of the top-level configuration",
			agent: &FluentbitForwarder{},
			want: map[string]string{
				"input-custom.conf":  "forwarder-input",
				"filter-custom.conf": "forwarder-filter",
			},
		},
		{
			name:  "the aggregator renders the custom sections of its own configuration",
			agent: &FluentbitAggregator{},
			want: map[string]string{
				"filter-custom.conf": "aggregator-filter",
				"output-custom.conf": "aggregator-output",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.agent.UpdateCustomConfiguration(map[string]string{}, newFluentbitHACR())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateCustomConfiguration() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestGetOutputFileName(t *testing.T) {
	t.Parallel()

	if got := (&Fluentbit{}).GetOutputFileName(); got != "output-log" {
		t.Fatalf("Fluentbit output file = %q, want %q", got, "output-log")
	}
	if got := (&Fluentd{}).GetOutputFileName(); got != "fake-fluent.log" {
		t.Fatalf("Fluentd output file = %q, want %q", got, "fake-fluent.log")
	}
	if got := (&FluentbitAggregator{}).GetOutputFileName(); got != "output-log" {
		t.Fatalf("FluentbitAggregator output file = %q, want %q", got, "output-log")
	}
}

func newFluentbitCR() *loggingService.LoggingService {
	return &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				CustomInputConf:  "input",
				CustomFilterConf: "filter",
				CustomOutputConf: "output",
			},
		},
	}
}

func newFluentbitHACR() *loggingService.LoggingService {
	return &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: &loggingService.Fluentbit{
				CustomInputConf:  "forwarder-input",
				CustomFilterConf: "forwarder-filter",
				Aggregator: &loggingService.FluentbitAggregator{
					CustomFilterConf: "aggregator-filter",
					CustomOutputConf: "aggregator-output",
				},
			},
		},
	}
}

func newFluentdCR() *loggingService.LoggingService {
	return &loggingService.LoggingService{
		Spec: loggingService.LoggingServiceSpec{
			Fluentd: &loggingService.Fluentd{
				CustomInputConf:  "input",
				CustomFilterConf: "filter",
				CustomOutputConf: "output",
			},
		},
	}
}
