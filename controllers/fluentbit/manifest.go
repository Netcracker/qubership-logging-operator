package fluentbit

import (
	"embed"
	"fmt"
	"maps"
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

//go:embed  fluentbit.configmap/conf.d/*
var fluentbitConfigs embed.FS

func fluentbitDaemonSet(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*appsv1.DaemonSet, error) {
	if cr.Spec.Fluentbit != nil {
		daemonSet := appsv1.DaemonSet{}
		cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType
		fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.FluentbitDaemonSet), util.FluentbitDaemonSet, cr.ToParams())
		if err != nil {
			return nil, err
		}
		if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&daemonSet); err != nil {
			return nil, err
		}
		util.SetLabelsForWorkload(&daemonSet, &daemonSet.Spec.Template.Labels, util.LabelInput{
			Name:            daemonSet.GetName(),
			Component:       "fluentbit",
			Instance:        util.GetInstanceLabel(daemonSet.GetName(), daemonSet.GetNamespace()),
			Version:         util.GetTagFromImage(cr.Spec.Fluentbit.DockerImage),
			Technology:      "c",
			ComponentLabels: cr.Spec.Fluentbit.Labels,
		})
		if cr.Spec.Fluentbit.Annotations != nil {
			daemonSet.SetAnnotations(cr.Spec.Fluentbit.Annotations)
			daemonSet.Spec.Template.SetAnnotations(cr.Spec.Fluentbit.Annotations)
		}
		if cr.Spec.Fluentbit.NodeSelectorKey != "" && cr.Spec.Fluentbit.NodeSelectorValue != "" {
			daemonSet.Spec.Template.Spec.NodeSelector = map[string]string{cr.Spec.Fluentbit.NodeSelectorKey: cr.Spec.Fluentbit.NodeSelectorValue}
		}
		if len(strings.TrimSpace(cr.Spec.Fluentbit.PriorityClassName)) > 0 {
			daemonSet.Spec.Template.Spec.PriorityClassName = cr.Spec.Fluentbit.PriorityClassName
		}
		if cr.Spec.Fluentbit.Tolerations != nil {
			daemonSet.Spec.Template.Spec.Tolerations = cr.Spec.Fluentbit.Tolerations
		}
		if cr.Spec.Fluentbit.Affinity != nil {
			daemonSet.Spec.Template.Spec.Affinity = cr.Spec.Fluentbit.Affinity
		}
		if cr.Spec.Fluentbit.AdditionalVolumes != nil {
			daemonSet.Spec.Template.Spec.Volumes = append(daemonSet.Spec.Template.Spec.Volumes, cr.Spec.Fluentbit.AdditionalVolumes...)
		}
		if cr.Spec.Fluentbit.AdditionalVolumeMounts != nil {
			for it := range daemonSet.Spec.Template.Spec.Containers {
				c := &daemonSet.Spec.Template.Spec.Containers[it]
				if c.Name == "logging-fluentbit" {
					c.VolumeMounts = append(c.VolumeMounts, cr.Spec.Fluentbit.AdditionalVolumeMounts...)
				}
			}
		}
		return &daemonSet, nil
	} else {
		return nil, fmt.Errorf("fluentbit configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
}

func fluentbitService(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*corev1.Service, error) {
	if cr.Spec.Fluentbit == nil {
		return nil, fmt.Errorf("fluentbit configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	service := corev1.Service{}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType
	fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.FluentbitService), util.FluentbitService, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&service); err != nil {
		return nil, err
	}
	util.SetLabelsForResource(&service, util.LabelInput{
		Name:            service.GetName(),
		Component:       "fluentbit",
		ComponentLabels: cr.Spec.Fluentbit.Labels,
	}, nil)
	return &service, nil
}

// fluentbitConfigSecret renders the whole Fluent Bit configuration into a single
// Secret instead of a ConfigMap. Sensitive output credentials are inlined into
// the configuration from the transient Auth *Value fields (resolved by the
// controller), so that no sensitive data is exposed through environment variables.
func fluentbitConfigSecret(cr *loggingService.LoggingService, dynamicParameters util.DynamicParameters) (*corev1.Secret, error) {
	if cr.Spec.Fluentbit == nil {
		return nil, fmt.Errorf("fluentbit configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	cr.Spec.ContainerRuntimeType = dynamicParameters.ContainerRuntimeType

	// Get Fluent-bit config from fluentbit.configmap/conf.d files
	configData, err := util.DataFromDirectory(fluentbitConfigs, util.FluentbitConfigMapDirectory, cr.ToParams())

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
		Component:       "fluentbit",
		ComponentLabels: cr.Spec.Fluentbit.Labels,
	}, map[string]string{"k8s-app": "fluent-bit"})
	return &secret, nil
}
