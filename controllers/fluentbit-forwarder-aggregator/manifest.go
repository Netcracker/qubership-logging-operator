package fluentbit_forwarder_aggregator

import (
	"embed"
	"fmt"
	"strings"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

//go:embed  assets/*.yaml
var assets embed.FS

//go:embed  aggregator.configmap/conf.d/*
var aggregatorConfigs embed.FS

//go:embed  forwarder.configmap/conf.d/*
var forwarderConfigs embed.FS

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
		for scriptName, script := range cr.Spec.Fluentbit.CustomLuaScriptConf {
			configMapData[scriptName] = script
		}
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
		Component:       "fluentbit",
		Instance:        util.ForwarderFluentbitComponentName + "-" + cr.GetNamespace(),
		Version:         util.GetTagFromImage(cr.Spec.Fluentbit.DockerImage),
		ComponentLabels: cr.Spec.Fluentbit.Labels,
	}, map[string]string{"k8s-app": "fluent-bit"})
	return &configMap, nil
}

func forwarderDaemonSet(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*appsv1.DaemonSet, error) {
	if cr.Spec.Fluentbit != nil {
		ds := appsv1.DaemonSet{}
		cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType
		fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.ForwarderFluentbitDaemonSet), util.ForwarderFluentbitDaemonSet, cr.ToParams())
		if err != nil {
			return nil, err
		}
		if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&ds); err != nil {
			return nil, err
		}
		util.SetLabelsForWorkload(&ds, &ds.Spec.Template.Labels, util.LabelInput{
			Name:            ds.GetName(),
			Component:       "fluentbit",
			Instance:        util.GetInstanceLabel(ds.GetName(), ds.GetNamespace()),
			Version:         util.GetTagFromImage(cr.Spec.Fluentbit.DockerImage),
			ComponentLabels: cr.Spec.Fluentbit.Labels,
		})

		if cr.Spec.Fluentbit.Annotations != nil {
			ds.SetAnnotations(cr.Spec.Fluentbit.Annotations)
			ds.Spec.Template.SetAnnotations(cr.Spec.Fluentbit.Annotations)
		}
		if cr.Spec.Fluentbit.NodeSelectorKey != "" && cr.Spec.Fluentbit.NodeSelectorValue != "" {
			ds.Spec.Template.Spec.NodeSelector = map[string]string{cr.Spec.Fluentbit.NodeSelectorKey: cr.Spec.Fluentbit.NodeSelectorValue}
		}
		if len(strings.TrimSpace(cr.Spec.Fluentbit.PriorityClassName)) > 0 {
			ds.Spec.Template.Spec.PriorityClassName = cr.Spec.Fluentbit.PriorityClassName
		}
		if cr.Spec.Fluentbit.Tolerations != nil {
			ds.Spec.Template.Spec.Tolerations = cr.Spec.Fluentbit.Tolerations
		}
		if cr.Spec.Fluentbit.Affinity != nil {
			ds.Spec.Template.Spec.Affinity = cr.Spec.Fluentbit.Affinity
		}
		return &ds, nil
	} else {
		return nil, fmt.Errorf("fluentbit configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
}

func forwarderService(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*corev1.Service, error) {
	if cr.Spec.Fluentbit == nil || cr.Spec.Fluentbit.Aggregator == nil {
		return nil, fmt.Errorf("fluentbit or aggregator configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	service := corev1.Service{}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType
	fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.ForwarderFluentbitService), util.ForwarderFluentbitService, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&service); err != nil {
		return nil, err
	}
	util.SetLabelsForResource(&service, util.LabelInput{
		Name:            service.GetName(),
		Component:       "fluentbit",
		Instance:        util.GetInstanceLabel(service.GetName(), service.GetNamespace()),
		Version:         util.GetTagFromImage(cr.Spec.Fluentbit.DockerImage),
		ComponentLabels: cr.Spec.Fluentbit.Labels,
	}, nil)
	return &service, nil
}

func aggregatorConfigMap(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*corev1.ConfigMap, error) {
	if cr.Spec.Fluentbit == nil || cr.Spec.Fluentbit.Aggregator == nil {
		return nil, fmt.Errorf("fluentbit or aggregator configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	// Get Fluent-bit forwarder config from forwarder.configmap/conf.d files
	configMapData, err := util.DataFromDirectory(aggregatorConfigs, util.AggregatorFluentbitConfigMapDirectory, cr.ToParams())

	if err != nil {
		return nil, err
	}

	if cr.Spec.Fluentbit.Aggregator.Output != nil && cr.Spec.Fluentbit.Aggregator.Output.Loki != nil &&
		cr.Spec.Fluentbit.Aggregator.Output.Loki.Enabled && cr.Spec.Fluentbit.Aggregator.Output.Loki.LabelsMapping != "" {
		configMapData["loki-labels.json"] = cr.Spec.Fluentbit.Aggregator.Output.Loki.LabelsMapping
	}

	// Set custom filters from parameters
	if cr.Spec.Fluentbit.Aggregator.CustomFilterConf != "" {
		configMapData["filter-custom.conf"] = cr.Spec.Fluentbit.Aggregator.CustomFilterConf
	}

	// Set custom output from parameters
	if cr.Spec.Fluentbit.Aggregator.CustomOutputConf != "" {
		configMapData["output-custom.conf"] = cr.Spec.Fluentbit.Aggregator.CustomOutputConf
	}

	// Set custom scripts from parameters
	if cr.Spec.Fluentbit.Aggregator.CustomLuaScriptConf != nil {
		for scriptName, script := range cr.Spec.Fluentbit.Aggregator.CustomLuaScriptConf {
			configMapData[scriptName] = script
		}
	}

	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.AggregatorFluentbitComponentName,
			Namespace: cr.GetNamespace(),
		},
		Data: configMapData,
	}
	util.SetLabelsForResource(&configMap, util.LabelInput{
		Name:            util.AggregatorFluentbitComponentName,
		Component:       "fluentbit",
		Instance:        util.AggregatorFluentbitComponentName + "-" + cr.GetNamespace(),
		Version:         util.GetTagFromImage(cr.Spec.Fluentbit.Aggregator.DockerImage),
		ComponentLabels: cr.Spec.Fluentbit.Aggregator.Labels,
	}, map[string]string{"k8s-app": "fluent-bit"})
	return &configMap, nil
}

func aggregatorStatefulSet(cr *loggingService.LoggingService) (*appsv1.StatefulSet, error) {
	statefulSet := appsv1.StatefulSet{}
	fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.AggregatorFluentbitStatefulSet), util.AggregatorFluentbitStatefulSet, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&statefulSet); err != nil {
		return nil, err
	}
	if cr.Spec.Fluentbit.Aggregator != nil {
		util.SetLabelsForWorkload(&statefulSet, &statefulSet.Spec.Template.Labels, util.LabelInput{
			Name:            statefulSet.GetName(),
			Component:       "fluentbit",
			Instance:        util.GetInstanceLabel(statefulSet.GetName(), statefulSet.GetNamespace()),
			Version:         util.GetTagFromImage(cr.Spec.Fluentbit.Aggregator.DockerImage),
			ComponentLabels: cr.Spec.Fluentbit.Aggregator.Labels,
		})
		if cr.Spec.Fluentbit.Aggregator.Annotations != nil {
			statefulSet.SetAnnotations(cr.Spec.Fluentbit.Aggregator.Annotations)
			statefulSet.Spec.Template.SetAnnotations(cr.Spec.Fluentbit.Aggregator.Annotations)
		}
		if cr.Spec.Fluentbit.Aggregator.NodeSelectorKey != "" && cr.Spec.Fluentbit.Aggregator.NodeSelectorValue != "" {
			statefulSet.Spec.Template.Spec.NodeSelector = map[string]string{cr.Spec.Fluentbit.Aggregator.NodeSelectorKey: cr.Spec.Fluentbit.Aggregator.NodeSelectorValue}
		}
		if len(strings.TrimSpace(cr.Spec.Fluentbit.Aggregator.PriorityClassName)) > 0 {
			statefulSet.Spec.Template.Spec.PriorityClassName = cr.Spec.Fluentbit.Aggregator.PriorityClassName
		}
		if cr.Spec.Fluentbit.Aggregator.Tolerations != nil {
			statefulSet.Spec.Template.Spec.Tolerations = cr.Spec.Fluentbit.Aggregator.Tolerations
		}
		if cr.Spec.Fluentbit.Aggregator.Affinity != nil {
			statefulSet.Spec.Template.Spec.Affinity = cr.Spec.Fluentbit.Aggregator.Affinity
		}
	}
	return &statefulSet, nil
}

func aggregatorService(cr *loggingService.LoggingService) (*corev1.Service, error) {
	if cr.Spec.Fluentbit == nil || cr.Spec.Fluentbit.Aggregator == nil {
		return nil, fmt.Errorf("fluentbit or aggregator configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	service := corev1.Service{}
	fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.AggregatorFluentbitService), util.AggregatorFluentbitService, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&service); err != nil {
		return nil, err
	}
	util.SetLabelsForResource(&service, util.LabelInput{
		Name:            service.GetName(),
		Component:       "fluentbit",
		Instance:        util.GetInstanceLabel(service.GetName(), service.GetNamespace()),
		Version:         util.GetTagFromImage(cr.Spec.Fluentbit.Aggregator.DockerImage),
		ComponentLabels: cr.Spec.Fluentbit.Aggregator.Labels,
	}, nil)
	return &service, nil
}
