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

func (flb *Fluentd) GetOutputFileName() string {
	return "fake-fluent.log"
}

type FluentbitHA struct {
	Fluentbit
}

func (flb *FluentbitHA) UpdateCustomConfiguration(data map[string]string, cr *loggingService.LoggingService) map[string]string {
	data[inputCustomConf] = cr.Spec.Fluentbit.CustomInputConf
	data[filterCustomConf] = cr.Spec.Fluentbit.Aggregator.CustomFilterConf
	data[outputCustomConf] = cr.Spec.Fluentbit.Aggregator.CustomOutputConf
	return data
}
