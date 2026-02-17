package fluentd

import (
	"embed"
	"fmt"
	"strings"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

//go:embed  assets/*.yaml
var assets embed.FS

//go:embed  fluentd.configmap/conf.d/*
var configs embed.FS

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

	//Set parameters
	configMap.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"})
	configMap.SetName(util.FluentdComponentName)
	configMap.SetNamespace(cr.GetNamespace())
	configMap.Data = data

	util.SetLabelsForResource(&configMap, util.LabelInput{
		Name:            util.FluentdComponentName,
		Component:       "fluentd",
		Instance:        util.GetInstanceLabel(configMap.GetName(), configMap.GetNamespace()),
		Version:         util.GetTagFromImage(cr.Spec.Fluentd.DockerImage),
		ComponentLabels: cr.Spec.Fluentd.Labels,
	}, nil)
	return &configMap, nil
}

func fluentdDaemonSet(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*appsv1.DaemonSet, error) {
	daemonSet := appsv1.DaemonSet{}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType
	fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.FluentdDaemonSet), util.FluentdDaemonSet, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&daemonSet); err != nil {
		return nil, err
	}
	if cr.Spec.Fluentd != nil {
		util.SetLabelsForWorkload(&daemonSet, &daemonSet.Spec.Template.Labels, util.LabelInput{
			Name:            daemonSet.GetName(),
			Component:       "fluentd",
			Instance:        util.GetInstanceLabel(daemonSet.GetName(), daemonSet.GetNamespace()),
			Version:         util.GetTagFromImage(cr.Spec.Fluentd.DockerImage),
			ComponentLabels: cr.Spec.Fluentd.Labels,
		})
		if cr.Spec.Fluentd.Annotations != nil {
			daemonSet.SetAnnotations(cr.Spec.Fluentd.Annotations)
			daemonSet.Spec.Template.SetAnnotations(cr.Spec.Fluentd.Annotations)
		}
		if cr.Spec.Fluentd.NodeSelectorKey != "" && cr.Spec.Fluentd.NodeSelectorValue != "" {
			daemonSet.Spec.Template.Spec.NodeSelector = map[string]string{cr.Spec.Fluentd.NodeSelectorKey: cr.Spec.Fluentd.NodeSelectorValue}
		}
		if len(strings.TrimSpace(cr.Spec.Fluentd.PriorityClassName)) > 0 {
			daemonSet.Spec.Template.Spec.PriorityClassName = cr.Spec.Fluentd.PriorityClassName
		}
		if cr.Spec.Fluentd.Tolerations != nil {
			daemonSet.Spec.Template.Spec.Tolerations = cr.Spec.Fluentd.Tolerations
		}
		if cr.Spec.Fluentd.Affinity != nil {
			daemonSet.Spec.Template.Spec.Affinity = cr.Spec.Fluentd.Affinity
		}
		if cr.Spec.Fluentd.AdditionalVolumes != nil {
			daemonSet.Spec.Template.Spec.Volumes = append(daemonSet.Spec.Template.Spec.Volumes, cr.Spec.Fluentd.AdditionalVolumes...)
		}
		if cr.Spec.Fluentd.AdditionalVolumeMounts != nil {
			for it := range daemonSet.Spec.Template.Spec.Containers {
				c := &daemonSet.Spec.Template.Spec.Containers[it]
				if c.Name == "logging-fluentd" {
					c.VolumeMounts = append(c.VolumeMounts, cr.Spec.Fluentd.AdditionalVolumeMounts...)
				}
			}
		}
	}
	return &daemonSet, nil
}

func fluentdService(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*corev1.Service, error) {
	if cr.Spec.Fluentd == nil {
		return nil, fmt.Errorf("fluentd configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	service := corev1.Service{}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType
	fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.FluentdServiceTemplate), util.FluentdServiceTemplate, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&service); err != nil {
		return nil, err
	}
	util.SetLabelsForResource(&service, util.LabelInput{
		Name:            service.GetName(),
		Component:       "fluentd",
		Instance:        util.GetInstanceLabel(service.GetName(), service.GetNamespace()),
		Version:         util.GetTagFromImage(cr.Spec.Fluentd.DockerImage),
		ComponentLabels: cr.Spec.Fluentd.Labels,
	}, nil)
	return &service, nil
}
