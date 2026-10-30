package fluentd

import (
	"embed"
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed  fluentd.configmap/conf.d/*
var configs embed.FS

type outputCredentials struct {
	Loki util.AuthValues
	Http util.AuthValues
}

type fluentdTemplateParameters struct {
	loggingService.LoggingServiceParameters
	OutputCredentials outputCredentials
}

func fluentdConfigSecret(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters, credentials outputCredentials) (*corev1.Secret, error) {
	if cr.Spec.Fluentd == nil {
		return nil, fmt.Errorf("fluentd configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType
	parameters := fluentdTemplateParameters{
		LoggingServiceParameters: cr.ToParams(),
		OutputCredentials:        credentials,
	}
	data, err := util.DataFromDirectory(configs, util.FluentdConfigMapDirectory, parameters)
	if err != nil {
		return nil, err
	}

	data["input-custom.conf"] = cr.Spec.Fluentd.CustomInputConf
	data["filter-custom.conf"] = cr.Spec.Fluentd.CustomFilterConf
	data["output-custom.conf"] = cr.Spec.Fluentd.CustomOutputConf

	if output := cr.Spec.Fluentd.Output; output != nil && output.Loki != nil &&
		output.Loki.Enabled && output.Loki.Auth != nil && output.Loki.Auth.Token != nil {
		data["loki-auth-token"] = credentials.Loki.Token
	}

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.FluentdComponentName,
			Namespace: cr.GetNamespace(),
		},
		Type: corev1.SecretTypeOpaque,
		Data: util.StringMapToByteMap(data),
	}
	util.SetLabelsForResource(&secret, util.LabelInput{
		Name:            util.FluentdComponentName,
		Component:       util.ComponentBackend,
		ComponentLabels: cr.Spec.Fluentd.Labels,
	}, nil)
	return &secret, nil
}
