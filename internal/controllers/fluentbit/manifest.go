package fluentbit

import (
	"embed"
	"fmt"
	"maps"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed  fluentbit.configmap/conf.d/*
var fluentbitConfigs embed.FS

// fluentbitConfigMap builds the FluentBit ConfigMap from the embedded conf.d directory
// plus per-CR overrides (custom input/filter/output/lua, Loki labels mapping). The
// DaemonSet and Service are constructed in
// internal/reconciler/factory/build/fluentbit; this file is intentionally limited to
// ConfigMap logic per the Stage 2 plan.
func fluentbitConfigMap(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*corev1.ConfigMap, error) {
	if cr.Spec.Fluentbit == nil {
		return nil, fmt.Errorf("fluentbit configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType

	configMapData, err := util.DataFromDirectory(fluentbitConfigs, util.FluentbitConfigMapDirectory, cr.ToParams())
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
	if cr.Spec.Fluentbit.CustomOutputConf != "" {
		configMapData["output-custom.conf"] = cr.Spec.Fluentbit.CustomOutputConf
	}
	if cr.Spec.Fluentbit.Output != nil && cr.Spec.Fluentbit.Output.Loki != nil &&
		cr.Spec.Fluentbit.Output.Loki.Enabled && cr.Spec.Fluentbit.Output.Loki.LabelsMapping != "" {
		configMapData["loki-labels.json"] = cr.Spec.Fluentbit.Output.Loki.LabelsMapping
	}

	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.FluentbitComponentName,
			Namespace: cr.GetNamespace(),
		},
		Data: configMapData,
	}
	util.SetLabelsForResource(&configMap, util.LabelInput{
		Name:            util.FluentbitComponentName,
		Component:       "fluentbit",
		ComponentLabels: cr.Spec.Fluentbit.Labels,
	}, map[string]string{"k8s-app": "fluent-bit"})
	return &configMap, nil
}
