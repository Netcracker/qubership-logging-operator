package agent

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
)

type Agent interface {
	UpdateCustomConfiguration(data map[string]string, cr *loggingService.LoggingService) map[string]string
	GetOutputFileName() string
}

type Fluentbit struct {
}

func (flb *Fluentbit) UpdateCustomConfiguration(data map[string]string, cr *loggingService.LoggingService) map[string]string {
	data["input-custom.conf"] = cr.Spec.Fluentbit.CustomInputConf
	data["filter-custom.conf"] = cr.Spec.Fluentbit.CustomFilterConf
	data["output-custom.conf"] = cr.Spec.Fluentbit.CustomOutputConf
	return data
}

func (flb *Fluentbit) GetOutputFileName() string {
	return "output-log"
}

type Fluentd struct {
}

func (flb *Fluentd) UpdateCustomConfiguration(data map[string]string, cr *loggingService.LoggingService) map[string]string {
	data["input-custom.conf"] = cr.Spec.Fluentd.CustomInputConf
	data["filter-custom.conf"] = cr.Spec.Fluentd.CustomFilterConf
	data["output-custom.conf"] = cr.Spec.Fluentd.CustomOutputConf
	return data
}

func (flb *Fluentd) GetOutputFileName() string {
	return "fake-fluent.log"
}

type FluentbitHA struct {
	Fluentbit
}

func (flb *FluentbitHA) UpdateCustomConfiguration(data map[string]string, cr *loggingService.LoggingService) map[string]string {
	data["input-custom.conf"] = cr.Spec.Fluentbit.CustomInputConf
	data["filter-custom.conf"] = cr.Spec.Fluentbit.Aggregator.CustomFilterConf
	data["output-custom.conf"] = cr.Spec.Fluentbit.Aggregator.CustomOutputConf
	return data
}
