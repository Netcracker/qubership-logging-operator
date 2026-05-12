package graylog

import (
	"embed"
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//go:embed  config/*
var configs embed.FS

// graylogConfigMap builds the Graylog ConfigMap from the embedded config/configmap
// directory. ServiceAccount, StatefulSet, Service, and MongoDB upgrade Jobs migrated
// to internal/reconciler/factory/build/graylog in Stage 5.
func graylogConfigMap(cr *loggingService.LoggingService) (*corev1.ConfigMap, error) {
	if cr.Spec.Graylog == nil {
		return nil, fmt.Errorf("graylog configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	configMap := corev1.ConfigMap{}
	data, err := util.DataFromDirectory(configs, util.GraylogConfigMapDirectory, cr.ToParams())
	if err != nil {
		return nil, err
	}

	configMap.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"})
	configMap.SetName(util.GraylogComponentName)
	configMap.SetNamespace(cr.GetNamespace())
	configMap.Data = data
	util.SetLabelsForResource(&configMap, util.LabelInput{
		Name:            util.GraylogComponentName,
		Component:       "graylog",
		ComponentLabels: cr.Spec.Graylog.Labels,
	}, nil)
	return &configMap, nil
}
