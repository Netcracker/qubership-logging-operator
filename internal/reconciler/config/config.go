// Package config holds operator-wide code-level defaults consulted by factory builders
// when the LoggingService CR (or Helm-supplied values rendered into it) leaves a field
// unset. User-supplied values always win; values here are the fallback.
//
// Each component has its own per-component default struct (FluentbitDefaults,
// GraylogDefaults, etc.) — populated incrementally as stages migrate components from
// YAML assets to Go factories. Initial values mirror today's
// charts/qubership-logging-operator/values.yaml + values.schema.json so existing
// deployments are behavior-preserving.
package config

// Defaults aggregates per-component defaults. Returned by Get() and threaded through
// reconcilers and factories.
type Defaults struct {
	EventsReader EventsReaderDefaults
	Fluentbit    FluentbitDefaults
	Fluentd      FluentdDefaults
	Graylog      GraylogDefaults
	Aggregator   FluentbitAggregatorDefaults
}

// Get returns a fully-populated Defaults struct. Stage 0 returns an empty struct;
// later stages populate per-component sub-structs as they are migrated.
func Get() *Defaults {
	return &Defaults{
		EventsReader: defaultEventsReader(),
		Fluentbit:    defaultFluentbit(),
		Fluentd:      defaultFluentd(),
		Graylog:      defaultGraylog(),
		Aggregator:   defaultAggregator(),
	}
}
