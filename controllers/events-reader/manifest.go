package events_reader

import (
	"embed"
	"fmt"
	"strings"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

//go:embed  assets/*.yaml
var assets embed.FS

func eventsReaderDeployment(cr *loggingService.LoggingService) (*appsv1.Deployment, error) {
	deployment := appsv1.Deployment{}
	fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.EventsReaderDeployment), util.EventsReaderDeployment, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&deployment); err != nil {
		return nil, err
	}
	if cr.Spec.CloudEventsReader != nil {
		util.SetLabelsForWorkload(&deployment, &deployment.Spec.Template.Labels, util.LabelInput{
			Name:            deployment.GetName(),
			Component:       "events-reader",
			Instance:        util.GetInstanceLabel(deployment.GetName(), deployment.GetNamespace()),
			Version:         util.GetTagFromImage(cr.Spec.CloudEventsReader.DockerImage),
			ComponentLabels: cr.Spec.CloudEventsReader.Labels,
		})
		if cr.Spec.CloudEventsReader.Annotations != nil {
			deployment.SetAnnotations(cr.Spec.CloudEventsReader.Annotations)
			deployment.Spec.Template.SetAnnotations(cr.Spec.CloudEventsReader.Annotations)
		}

		if cr.Spec.CloudEventsReader.Affinity != nil {
			deployment.Spec.Template.Spec.Affinity = cr.Spec.CloudEventsReader.Affinity
		}

		if len(strings.TrimSpace(cr.Spec.CloudEventsReader.PriorityClassName)) > 0 {
			deployment.Spec.Template.Spec.PriorityClassName = cr.Spec.CloudEventsReader.PriorityClassName
		}
	}

	return &deployment, nil
}

func eventsReaderService(cr *loggingService.LoggingService) (*corev1.Service, error) {
	if cr.Spec.CloudEventsReader == nil {
		return nil, fmt.Errorf("cloud events reader configuration in Logging Service %s is nil in the namespace %s", cr.GetName(), cr.GetNamespace())
	}
	service := corev1.Service{}
	fileContent, err := util.ParseTemplate(util.MustAssetReader(assets, util.EventsReaderService), util.EventsReaderService, cr.ToParams())
	if err != nil {
		return nil, err
	}
	if err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fileContent), util.BufferSize).Decode(&service); err != nil {
		return nil, err
	}
	util.SetLabelsForResource(&service, util.LabelInput{
		Name:            service.GetName(),
		Component:       "events-reader",
		Instance:        util.GetInstanceLabel(service.GetName(), service.GetNamespace()),
		Version:         util.GetTagFromImage(cr.Spec.CloudEventsReader.DockerImage),
		ComponentLabels: cr.Spec.CloudEventsReader.Labels,
	}, nil)
	return &service, nil
}
