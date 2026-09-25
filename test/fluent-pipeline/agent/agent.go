package agent

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
)

// Names of the rendered configuration files that carry the custom sections of the custom resource.
const (
	inputCustomConf  = "input-custom.conf"
	filterCustomConf = "filter-custom.conf"
	outputCustomConf = "output-custom.conf"
)

type Agent interface {
	UpdateCustomConfiguration(data map[string]string, cr *loggingService.LoggingService) map[string]string
	GetOutputFileName() string
}

type Fluentbit struct {
}

func (flb *Fluentbit) UpdateCustomConfiguration(data map[string]string, cr *loggingService.LoggingService) map[string]string {
	data[inputCustomConf] = cr.Spec.Fluentbit.CustomInputConf
	data[filterCustomConf] = cr.Spec.Fluentbit.CustomFilterConf
	data[outputCustomConf] = cr.Spec.Fluentbit.CustomOutputConf
	return data
}

// String names the agent in the report of a run.
func (flb *Fluentbit) String() string {
	return "Fluent Bit"
}

func (flb *Fluentbit) GetOutputFileName() string {
	return "output-log"
}

type Fluentd struct {
}

func (flb *Fluentd) UpdateCustomConfiguration(data map[string]string, cr *loggingService.LoggingService) map[string]string {
	data[inputCustomConf] = cr.Spec.Fluentd.CustomInputConf
	data[filterCustomConf] = cr.Spec.Fluentd.CustomFilterConf
	data[outputCustomConf] = cr.Spec.Fluentd.CustomOutputConf
	return data
}

func (flb *Fluentd) String() string {
	return "Fluentd"
}

func (flb *Fluentd) GetOutputFileName() string {
	return "fake-fluent.log"
}

// The two roles of the HA deployment read different fields of the custom resource, the way the
// operator fills their config maps: the forwarder takes the top-level custom sections and the
// aggregator its own. One renderer for both would hand a role a section it never receives.

type FluentbitForwarder struct {
	Fluentbit
}

func (flb *FluentbitForwarder) String() string {
	return "Fluent Bit forwarder"
}

// UpdateCustomConfiguration fills the sections the forwarder config map carries. Its configuration
// includes no custom output: the forwarder always forwards to the aggregator.
func (flb *FluentbitForwarder) UpdateCustomConfiguration(data map[string]string, cr *loggingService.LoggingService) map[string]string {
	data[inputCustomConf] = cr.Spec.Fluentbit.CustomInputConf
	data[filterCustomConf] = cr.Spec.Fluentbit.CustomFilterConf
	return data
}

type FluentbitAggregator struct {
	Fluentbit
}

func (flb *FluentbitAggregator) String() string {
	return "Fluent Bit aggregator"
}

// UpdateCustomConfiguration fills the sections the aggregator config map carries. Its
// configuration includes no custom input: the aggregator reads from the forwarder.
func (flb *FluentbitAggregator) UpdateCustomConfiguration(data map[string]string, cr *loggingService.LoggingService) map[string]string {
	data[filterCustomConf] = cr.Spec.Fluentbit.Aggregator.CustomFilterConf
	data[outputCustomConf] = cr.Spec.Fluentbit.Aggregator.CustomOutputConf
	return data
}
