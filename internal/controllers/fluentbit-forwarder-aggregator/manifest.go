package fluentbit_forwarder_aggregator

import (
	"embed"
	"fmt"
	"maps"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed  aggregator.configmap/conf.d/*
var aggregatorConfigs embed.FS

//go:embed  forwarder.configmap/conf.d/*
var forwarderConfigs embed.FS

type aggregatorOutputCredentials struct {
	Loki util.AuthValues
	Http util.AuthValues
	Otel util.AuthValues
}

type aggregatorTemplateParameters struct {
	loggingService.LoggingServiceParameters
	OutputCredentials aggregatorOutputCredentials
}

func forwarderConfigMap(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*corev1.ConfigMap, error) {
	if cr.Spec.Fluentbit == nil {
		return nil, fmt.Errorf("fluentbit configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType

	// Get Fluent-bit forwarder config from forwarder.configmap/conf.d files
	configMapData, err := util.DataFromDirectory(forwarderConfigs, util.ForwarderFluentbitConfigMapDirectory, cr.ToParams())

	if err != nil {
		return nil, err
	}

	// Set custom input from parameters
	if cr.Spec.Fluentbit.CustomInputConf != "" {
		configMapData["input-custom.conf"] = cr.Spec.Fluentbit.CustomInputConf
	}

	// Set custom filters from parameters
	if cr.Spec.Fluentbit.CustomFilterConf != "" {
		configMapData["filter-custom.conf"] = cr.Spec.Fluentbit.CustomFilterConf
	}

	// Set custom scripts from parameters
	if cr.Spec.Fluentbit.CustomLuaScriptConf != nil {
		maps.Copy(configMapData, cr.Spec.Fluentbit.CustomLuaScriptConf)
	}

	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.ForwarderFluentbitComponentName,
			Namespace: cr.GetNamespace(),
		},
		Data: configMapData,
	}
	util.SetLabelsForResource(&configMap, util.LabelInput{
		Name:            util.ForwarderFluentbitComponentName,
		Component:       util.ComponentBackend,
		ComponentLabels: cr.Spec.Fluentbit.Labels,
	}, map[string]string{"k8s-app": "fluent-bit"})
	return &configMap, nil
}

func aggregatorConfigSecret(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters, credentials aggregatorOutputCredentials) (*corev1.Secret, error) {
	if cr.Spec.Fluentbit == nil || cr.Spec.Fluentbit.Aggregator == nil {
		return nil, fmt.Errorf("fluentbit or aggregator configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType
	parameters := aggregatorTemplateParameters{
		LoggingServiceParameters: cr.ToParams(),
		OutputCredentials:        credentials,
	}
	// Get Fluent-bit aggregator config from aggregator.configmap/conf.d files
	configData, err := util.DataFromDirectory(aggregatorConfigs, util.AggregatorFluentbitConfigMapDirectory, parameters)

	if err != nil {
		return nil, err
	}

	if cr.Spec.Fluentbit.Aggregator.Output != nil && cr.Spec.Fluentbit.Aggregator.Output.Loki != nil &&
		cr.Spec.Fluentbit.Aggregator.Output.Loki.Enabled && cr.Spec.Fluentbit.Aggregator.Output.Loki.LabelsMapping != "" {
		configData["loki-labels.json"] = cr.Spec.Fluentbit.Aggregator.Output.Loki.LabelsMapping
	}

	// Set custom filters from parameters
	if cr.Spec.Fluentbit.Aggregator.CustomFilterConf != "" {
		configData["filter-custom.conf"] = cr.Spec.Fluentbit.Aggregator.CustomFilterConf
	}

	// Set custom output from parameters
	if cr.Spec.Fluentbit.Aggregator.CustomOutputConf != "" {
		configData["output-custom.conf"] = cr.Spec.Fluentbit.Aggregator.CustomOutputConf
	}

	// Set custom scripts from parameters
	if cr.Spec.Fluentbit.Aggregator.CustomLuaScriptConf != nil {
		maps.Copy(configData, cr.Spec.Fluentbit.Aggregator.CustomLuaScriptConf)
	}

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.AggregatorFluentbitComponentName,
			Namespace: cr.GetNamespace(),
		},
		Type: corev1.SecretTypeOpaque,
		Data: util.StringMapToByteMap(configData),
	}
	util.SetLabelsForResource(&secret, util.LabelInput{
		Name:            util.AggregatorFluentbitComponentName,
		Component:       util.ComponentBackend,
		ComponentLabels: cr.Spec.Fluentbit.Aggregator.Labels,
	}, map[string]string{"k8s-app": "fluent-bit"})
	return &secret, nil
}
