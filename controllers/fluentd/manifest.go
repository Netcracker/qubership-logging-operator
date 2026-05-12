package fluentd

import (
	"embed"
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//go:embed  fluentd.configmap/conf.d/*
var configs embed.FS

// fluentdConfigMap builds the FluentD ConfigMap from the embedded conf.d directory and
// CR overrides (custom input/filter/output). The DaemonSet and Service are constructed
// in internal/reconciler/factory/build/fluentd; this file is intentionally limited to
// ConfigMap logic per the Stage 3 plan.
func fluentdConfigMap(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*corev1.ConfigMap, error) {
	if cr.Spec.Fluentd == nil {
		return nil, fmt.Errorf("fluentd configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	configMap := corev1.ConfigMap{}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType
	data, err := util.DataFromDirectory(configs, util.FluentdConfigMapDirectory, cr.ToParams())
	if err != nil {
		return nil, err
	}

	data["input-custom.conf"] = cr.Spec.Fluentd.CustomInputConf
	data["filter-custom.conf"] = cr.Spec.Fluentd.CustomFilterConf
	data["output-custom.conf"] = cr.Spec.Fluentd.CustomOutputConf

	configMap.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"})
	configMap.SetName(util.FluentdComponentName)
	configMap.SetNamespace(cr.GetNamespace())
	configMap.Data = data

	util.SetLabelsForResource(&configMap, util.LabelInput{
		Name:            util.FluentdComponentName,
		Component:       "fluentd",
		ComponentLabels: cr.Spec.Fluentd.Labels,
	}, nil)
	return &configMap, nil
}
