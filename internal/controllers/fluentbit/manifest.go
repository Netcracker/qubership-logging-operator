package fluentbit

import (
	"embed"
	"fmt"
	"maps"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed  fluentbit.configmap/conf.d/*
var fluentbitConfigs embed.FS

type outputCredentials struct {
	Loki util.AuthValues
	Http util.AuthValues
	Otel util.AuthValues
}

type fluentbitTemplateParameters struct {
	loggingService.LoggingServiceParameters
	OutputCredentials outputCredentials
}

// fluentbitConfigSecret renders the whole Fluent Bit configuration into a single
// Secret instead of a ConfigMap. Sensitive output credentials are resolved by the
// controller and passed in explicitly via credentials, so that no sensitive data
// is exposed through environment variables or persisted on the Logging Service CR.
func fluentbitConfigSecret(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters, credentials outputCredentials) (*corev1.Secret, error) {
	if cr.Spec.Fluentbit == nil {
		return nil, fmt.Errorf("fluentbit configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType
	parameters := fluentbitTemplateParameters{
		LoggingServiceParameters: cr.ToParams(),
		OutputCredentials:        credentials,
	}

	// Get Fluent-bit config from fluentbit.configmap/conf.d files
	configData, err := util.DataFromDirectory(fluentbitConfigs, util.FluentbitConfigMapDirectory, parameters)

	if err != nil {
		return nil, err
	}

	// Set custom input from parameters
	if cr.Spec.Fluentbit.CustomInputConf != "" {
		configData["input-custom.conf"] = cr.Spec.Fluentbit.CustomInputConf
	}

	// Set custom filters from parameters
	if cr.Spec.Fluentbit.CustomFilterConf != "" {
		configData["filter-custom.conf"] = cr.Spec.Fluentbit.CustomFilterConf
	}

	// Set custom scripts from parameters
	if cr.Spec.Fluentbit.CustomLuaScriptConf != nil {
		maps.Copy(configData, cr.Spec.Fluentbit.CustomLuaScriptConf)
	}

	// Set custom output from parameters
	if cr.Spec.Fluentbit.CustomOutputConf != "" {
		configData["output-custom.conf"] = cr.Spec.Fluentbit.CustomOutputConf
	}

	if cr.Spec.Fluentbit.Output != nil && cr.Spec.Fluentbit.Output.Loki != nil &&
		cr.Spec.Fluentbit.Output.Loki.Enabled && cr.Spec.Fluentbit.Output.Loki.LabelsMapping != "" {
		configData["loki-labels.json"] = cr.Spec.Fluentbit.Output.Loki.LabelsMapping
	}

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.FluentbitComponentName,
			Namespace: cr.GetNamespace(),
		},
		Type: corev1.SecretTypeOpaque,
		Data: util.StringMapToByteMap(configData),
	}
	util.SetLabelsForResource(&secret, util.LabelInput{
		Name:            util.FluentbitComponentName,
		Component:       util.ComponentBackend,
		ComponentLabels: cr.Spec.Fluentbit.Labels,
	}, map[string]string{"k8s-app": "fluent-bit"})
	return &secret, nil
}
