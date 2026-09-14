package agent

import (
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

func TestFluentbitHAUpdateCustomConfigurationUsesAggregatorConfig(t *testing.T) {
	t.Parallel()

	cr := newFluentbitHACR()
	got := (&FluentbitHA{}).UpdateCustomConfiguration(map[string]string{}, cr)

	if got["filter-custom.conf"] == "" || got["output-custom.conf"] == "" {
		t.Fatalf("expected fluentbit-ha aggregator configuration to be populated, got %#v", got)
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
	if got := (&FluentbitHA{}).GetOutputFileName(); got != "output-log" {
		t.Fatalf("FluentbitHA output file = %q, want %q", got, "output-log")
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
				CustomInputConf: "input",
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
