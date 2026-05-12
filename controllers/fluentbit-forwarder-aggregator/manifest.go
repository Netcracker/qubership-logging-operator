package fluentbit_forwarder_aggregator

import (
	"embed"
	"fmt"
	"maps"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed  aggregator.configmap/conf.d/*
var aggregatorConfigs embed.FS

//go:embed  forwarder.configmap/conf.d/*
var forwarderConfigs embed.FS

// forwarderConfigMap builds the forwarder ConfigMap. DaemonSet + Service migrated to
// internal/reconciler/factory/build/forwarderaggregator in Stage 4.
func forwarderConfigMap(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*corev1.ConfigMap, error) {
	if cr.Spec.Fluentbit == nil {
		return nil, fmt.Errorf("fluentbit configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType

	configMapData, err := util.DataFromDirectory(forwarderConfigs, util.ForwarderFluentbitConfigMapDirectory, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if cr.Spec.Fluentbit.CustomInputConf != "" {
		configMapData["input-custom.conf"] = cr.Spec.Fluentbit.CustomInputConf
	}
	if cr.Spec.Fluentbit.CustomFilterConf != "" {
		configMapData["filter-custom.conf"] = cr.Spec.Fluentbit.CustomFilterConf
	}
	if cr.Spec.Fluentbit.CustomLuaScriptConf != nil {
		maps.Copy(configMapData, cr.Spec.Fluentbit.CustomLuaScriptConf)
	}

	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.ForwarderFluentbitComponentName,
			Namespace: cr.GetNamespace(),
		},
		Data: configMapData,
	}
	util.SetLabelsForResource(&configMap, util.LabelInput{
		Name:            util.ForwarderFluentbitComponentName,
		Component:       "fluentbit",
		ComponentLabels: cr.Spec.Fluentbit.Labels,
	}, map[string]string{"k8s-app": "fluent-bit"})
	return &configMap, nil
}

// aggregatorConfigMap builds the aggregator ConfigMap. StatefulSet + Service migrated
// to internal/reconciler/factory/build/forwarderaggregator in Stage 4.
func aggregatorConfigMap(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*corev1.ConfigMap, error) {
	if cr.Spec.Fluentbit == nil || cr.Spec.Fluentbit.Aggregator == nil {
		return nil, fmt.Errorf("fluentbit or aggregator configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType

	configMapData, err := util.DataFromDirectory(aggregatorConfigs, util.AggregatorFluentbitConfigMapDirectory, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if cr.Spec.Fluentbit.Aggregator.Output != nil && cr.Spec.Fluentbit.Aggregator.Output.Loki != nil &&
		cr.Spec.Fluentbit.Aggregator.Output.Loki.Enabled && cr.Spec.Fluentbit.Aggregator.Output.Loki.LabelsMapping != "" {
		configMapData["loki-labels.json"] = cr.Spec.Fluentbit.Aggregator.Output.Loki.LabelsMapping
	}
	if cr.Spec.Fluentbit.Aggregator.CustomFilterConf != "" {
		configMapData["filter-custom.conf"] = cr.Spec.Fluentbit.Aggregator.CustomFilterConf
	}
	if cr.Spec.Fluentbit.Aggregator.CustomOutputConf != "" {
		configMapData["output-custom.conf"] = cr.Spec.Fluentbit.Aggregator.CustomOutputConf
	}
	if cr.Spec.Fluentbit.Aggregator.CustomLuaScriptConf != nil {
		maps.Copy(configMapData, cr.Spec.Fluentbit.Aggregator.CustomLuaScriptConf)
	}

	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.AggregatorFluentbitComponentName,
			Namespace: cr.GetNamespace(),
		},
		Data: configMapData,
	}
	util.SetLabelsForResource(&configMap, util.LabelInput{
		Name:            util.AggregatorFluentbitComponentName,
		Component:       "fluentbit",
		ComponentLabels: cr.Spec.Fluentbit.Aggregator.Labels,
	}, map[string]string{"k8s-app": "fluent-bit"})
	return &configMap, nil
}
