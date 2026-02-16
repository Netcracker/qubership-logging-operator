package graylog

import (
	"embed"
	"fmt"
	"strings"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

//go:embed  assets/*.yaml
var assets embed.FS

//go:embed  config/*
var configs embed.FS

func graylogServiceAccount(cr *loggingService.LoggingService) (*corev1.ServiceAccount, error) {
	if cr.Spec.Graylog == nil {
		return nil, fmt.Errorf("graylog configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	sa := corev1.ServiceAccount{}
	fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.GraylogServiceAccount), util.GraylogServiceAccount, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&sa); err != nil {
		return nil, err
	}
	util.SetLabelsForResource(&sa, util.LabelInput{
		Name:            sa.GetName(),
		Component:       "graylog",
		Instance:        util.GetInstanceLabel(sa.GetName(), sa.GetNamespace()),
		Version:         util.GetTagFromImage(cr.Spec.Graylog.DockerImage),
		ComponentLabels: cr.Spec.Graylog.Labels,
	}, nil)
	return &sa, nil
}

func graylogConfigMap(cr *loggingService.LoggingService) (*corev1.ConfigMap, error) {
	if cr.Spec.Graylog == nil {
		return nil, fmt.Errorf("graylog configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	configMap := corev1.ConfigMap{}
	data, err := util.DataFromDirectory(configs, util.GraylogConfigMapDirectory, cr.ToParams())
	if err != nil {
		return nil, err
	}

	//Set parameters
	configMap.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"})
	configMap.SetName(util.GraylogComponentName)
	configMap.SetNamespace(cr.GetNamespace())
	configMap.Data = data
	util.SetLabelsForResource(&configMap, util.LabelInput{
		Name:            util.GraylogComponentName,
		Component:       "graylog",
		Instance:        util.GetInstanceLabel(configMap.GetName(), configMap.GetNamespace()),
		Version:         util.GetTagFromImage(cr.Spec.Graylog.DockerImage),
		ComponentLabels: cr.Spec.Graylog.Labels,
	}, nil)
	return &configMap, nil
}

func graylogMongoUpgradeJob(cr *loggingService.LoggingService, assetPath string) (*batchv1.Job, error) {
	if cr.Spec.Graylog == nil {
		return nil, fmt.Errorf("graylog configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	job := batchv1.Job{}
	fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, assetPath), assetPath, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&job); err != nil {
		return nil, err
	}
	util.SetLabelsForWorkload(&job, &job.Spec.Template.Labels, util.LabelInput{
		Name:            job.GetName(),
		Component:       "graylog",
		Instance:        util.GetInstanceLabel(job.GetName(), job.GetNamespace()),
		Version:         util.GetTagFromImage(job.Spec.Template.Spec.Containers[0].Image),
		ComponentLabels: cr.Spec.Graylog.Labels,
	})
	return &job, nil
}

func graylogStatefulset(cr *loggingService.LoggingService) (*appsv1.StatefulSet, error) {
	statefulset := appsv1.StatefulSet{}
	fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.GraylogStatefulset), util.GraylogStatefulset, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&statefulset); err != nil {
		return nil, err
	}
	if cr.Spec.Graylog != nil {
		util.SetLabelsForWorkload(&statefulset, &statefulset.Spec.Template.Labels, util.LabelInput{
			Name:            statefulset.GetName(),
			Component:       "graylog",
			Instance:        util.GetInstanceLabel(statefulset.GetName(), statefulset.GetNamespace()),
			Version:         util.GetTagFromImage(cr.Spec.Graylog.DockerImage),
			ComponentLabels: cr.Spec.Graylog.Labels,
		})
		if cr.Spec.Graylog.Annotations != nil {
			statefulset.SetAnnotations(cr.Spec.Graylog.Annotations)
			statefulset.Spec.Template.SetAnnotations(cr.Spec.Graylog.Annotations)
		}

		if cr.Spec.Graylog.Affinity != nil {
			statefulset.Spec.Template.Spec.Affinity = cr.Spec.Graylog.Affinity
		}

		if len(strings.TrimSpace(cr.Spec.Graylog.PriorityClassName)) > 0 {
			statefulset.Spec.Template.Spec.PriorityClassName = cr.Spec.Graylog.PriorityClassName
		}
	}
	return &statefulset, nil
}

func graylogService(cr *loggingService.LoggingService) (*corev1.Service, error) {
	if cr.Spec.Graylog == nil {
		return nil, fmt.Errorf("graylog configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	service := corev1.Service{}
	fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.GraylogService), util.GraylogService, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&service); err != nil {
		return nil, err
	}
	util.SetLabelsForResource(&service, util.LabelInput{
		Name:            service.GetName(),
		Component:       "graylog",
		Instance:        util.GetInstanceLabel(service.GetName(), service.GetNamespace()),
		Version:         util.GetTagFromImage(cr.Spec.Graylog.DockerImage),
		ComponentLabels: cr.Spec.Graylog.Labels,
	}, nil)
	return &service, nil
}
