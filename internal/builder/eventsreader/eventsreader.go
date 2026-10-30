// Package eventsreader builds the Deployment and Service for the CloudEventsReader
// component from the LoggingService CR plus code-level defaults. It replaces the
// embedded YAML template at controllers/events-reader/assets/.
package eventsreader

import (
	"context"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/reconcile"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ComponentName is the value used for app.kubernetes.io/component and as the
	// Deployment / Service / container name. Matches util.EventsReaderComponentName.
	ComponentName = "events-reader"
	// Technology is set on the pod-template app.kubernetes.io/technology label.
	Technology = "go"
	// EnvCloudURL is the env var the events-reader binary reads to learn the cluster URL.
	EnvCloudURL = "OPENSHIFT_URL"
)

// CreateOrUpdate reconciles the events-reader Deployment and Service against the
// cluster. Caller is responsible for gating on cr.Spec.CloudEventsReader.IsInstall().
func CreateOrUpdate(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService, cfg *config.Defaults) error {
	def := cfg.EventsReader
	deploy := buildDeployment(cr, def)
	if err := reconcile.Deployment(ctx, c, scheme, cr, deploy); err != nil {
		return err
	}
	return reconcile.Service(ctx, c, scheme, cr, buildService(cr))
}

// selector is the label selector linking Service → pods and Deployment → pods. Matches
// the original YAML asset (selector.matchLabels.name=events-reader).
func selector() map[string]string {
	return map[string]string{"name": ComponentName}
}

func buildDeployment(cr *loggingService.LoggingService, def config.EventsReaderDefaults) *appsv1.Deployment {
	spec := cr.Spec.CloudEventsReader
	image := build.FirstString(spec.DockerImage, def.Image)
	resources := spec.Resources
	if resources == nil {
		r := def.Resources.DeepCopy()
		resources = r
	}

	container := build.NewContainer(ComponentName, build.ContainerOpts{
		Image:           image,
		ImagePullPolicy: def.ImagePullPolicy,
		Command:         def.Command,
		Args:            spec.Args,
		Env:             []corev1.EnvVar{{Name: EnvCloudURL, Value: cr.Spec.CloudURL}},
		Ports:           []corev1.ContainerPort{{ContainerPort: def.Port, Protocol: corev1.ProtocolTCP}},
		Resources:       *resources,
		LivenessProbe:   def.LivenessProbe.DeepCopy(),
		ReadinessProbe:  def.ReadinessProbe.DeepCopy(),
		SecurityContext: containerSecurityContext(cr, def),
	})

	pod := corev1.PodSpec{
		ServiceAccountName: ComponentName,
		Containers:         []corev1.Container{container},
		NodeSelector:       nodeSelector(spec.NodeSelectorKey, spec.NodeSelectorValue),
		Affinity:           spec.Affinity,
		PriorityClassName:  spec.PriorityClassName,
	}

	maxSurge := def.MaxSurge
	maxUnavailable := def.MaxUnavailable
	deploy := build.NewDeployment(ComponentName, cr.GetNamespace(), ComponentName, build.DeploymentOpts{
		Selector: selector(),
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxSurge:       &maxSurge,
				MaxUnavailable: &maxUnavailable,
			},
		},
		PodSpec:        pod,
		PodAnnotations: spec.Annotations,
		Annotations:    spec.Annotations,
	})

	util.SetLabelsForWorkload(deploy, &deploy.Spec.Template.Labels, util.LabelInput{
		Name:            ComponentName,
		Component:       ComponentName,
		Instance:        util.GetInstanceLabel(ComponentName, cr.GetNamespace()),
		Version:         util.GetTagFromImage(image),
		Technology:      Technology,
		ComponentLabels: spec.Labels,
	})
	return deploy
}

func buildService(cr *loggingService.LoggingService) *corev1.Service {
	spec := cr.Spec.CloudEventsReader
	port := int32(8080)
	svc := build.NewService(ComponentName, cr.GetNamespace(), ComponentName, build.ServiceOpts{
		Type:     corev1.ServiceTypeClusterIP,
		Ports:    []corev1.ServicePort{{Name: ComponentName, Port: port, TargetPort: intstr.FromInt32(port), Protocol: corev1.ProtocolTCP}},
		Selector: selector(),
	})
	util.SetLabelsForResource(svc, util.LabelInput{
		Name:            ComponentName,
		Component:       ComponentName,
		ComponentLabels: spec.Labels,
	}, nil)
	return svc
}

func nodeSelector(key, value string) map[string]string {
	if key == "" || value == "" {
		return nil
	}
	return map[string]string{key: value}
}

// containerSecurityContext mirrors the original YAML conditional: when the cluster is
// not OpenShift, run the container as non-root user 1000; on OpenShift the SCC handles
// it, so leave the security context empty.
func containerSecurityContext(cr *loggingService.LoggingService, def config.EventsReaderDefaults) *corev1.SecurityContext {
	if cr.Spec.OpenshiftDeploy {
		return nil
	}
	if def.SecurityContext == nil {
		return nil
	}
	sc := *def.SecurityContext
	return &sc
}
