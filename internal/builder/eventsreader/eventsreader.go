// Package eventsreader builds the CloudEventsReader Deployment and Service from the
// LoggingService custom resource.
package eventsreader

import (
	"strings"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	name       = "events-reader"
	technology = "go"
	tmpVolume  = "tmp"
)

// selector links the Service and the Deployment to the events reader pods.
func selector() map[string]string {
	return map[string]string{"name": name}
}

// Deployment returns the CloudEventsReader Deployment. The caller checks that
// cr.Spec.CloudEventsReader is set.
func Deployment(cr *loggingService.LoggingService, cfg *defaults.Defaults) *appsv1.Deployment {
	spec := cr.Spec.CloudEventsReader
	def := cfg.EventsReader

	args := spec.Args
	if args == nil {
		args = []string{}
	}
	container := builder.NewContainer(name, builder.ContainerOpts{
		Image:           spec.DockerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         def.Command,
		Args:            args,
		Env:             []corev1.EnvVar{{Name: "OPENSHIFT_URL", Value: cr.Spec.CloudURL}},
		Ports:           []corev1.ContainerPort{{ContainerPort: def.Port, Protocol: corev1.ProtocolTCP}},
		Resources:       *builder.FirstNonNil(spec.Resources, &def.Resources).DeepCopy(),
		VolumeMounts:    []corev1.VolumeMount{{Name: tmpVolume, MountPath: "/tmp"}},
		LivenessProbe:   def.LivenessProbe.DeepCopy(),
		ReadinessProbe:  def.ReadinessProbe.DeepCopy(),
		SecurityContext: containerSecurityContext(cr.Spec.OpenshiftDeploy, def.RunAsID),
	})

	podSpec := corev1.PodSpec{
		ServiceAccountName: name,
		NodeSelector:       builder.NodeSelector(spec.NodeSelectorKey, spec.NodeSelectorValue),
		SecurityContext:    podSecurityContext(cr.Spec.OpenshiftDeploy, def.RunAsID),
		Volumes: []corev1.Volume{{
			Name:         tmpVolume,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: &def.TmpVolumeSize}},
		}},
		Containers: []corev1.Container{container},
		Affinity:   spec.Affinity,
	}
	if strings.TrimSpace(spec.PriorityClassName) != "" {
		podSpec.PriorityClassName = spec.PriorityClassName
	}

	maxSurge, maxUnavailable := def.RollingMaxSurge, def.RollingMaxUnavailable
	deployment := builder.NewDeployment(name, cr.GetNamespace(), builder.DeploymentOpts{
		Selector: selector(),
		Strategy: appsv1.DeploymentStrategy{
			Type:          appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{MaxSurge: &maxSurge, MaxUnavailable: &maxUnavailable},
		},
		PodSpec:        podSpec,
		Annotations:    spec.Annotations,
		PodAnnotations: spec.Annotations,
	})
	util.SetLabelsForWorkload(deployment, &deployment.Spec.Template.Labels, util.LabelInput{
		Name:            name,
		Component:       util.ComponentBackend,
		Instance:        util.GetInstanceLabel(name, cr.GetNamespace()),
		Version:         util.GetTagFromImage(spec.DockerImage),
		Technology:      technology,
		ComponentLabels: spec.Labels,
	})
	return deployment
}

// Service returns the CloudEventsReader Service. The caller checks that
// cr.Spec.CloudEventsReader is set.
func Service(cr *loggingService.LoggingService, cfg *defaults.Defaults) *corev1.Service {
	port := cfg.EventsReader.Port
	service := builder.NewService(name, cr.GetNamespace(), builder.ServiceOpts{
		Selector: selector(),
		Ports: []corev1.ServicePort{{
			Name:       name,
			Port:       port,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt32(port),
		}},
	})
	util.SetLabelsForResource(service, util.LabelInput{
		Name:            name,
		Component:       util.ComponentBackend,
		ComponentLabels: cr.Spec.CloudEventsReader.Labels,
	}, nil)
	return service
}

// podSecurityContext leaves runAsUser unset on OpenShift, where the security context
// constraint assigns the user.
func podSecurityContext(openshift bool, id int64) *corev1.PodSecurityContext {
	sc := &corev1.PodSecurityContext{
		RunAsNonRoot:   ptr.To(true),
		RunAsGroup:     ptr.To(id),
		SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
	}
	if !openshift {
		sc.RunAsUser = ptr.To(id)
	}
	return sc
}

func containerSecurityContext(openshift bool, id int64) *corev1.SecurityContext {
	sc := &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		ReadOnlyRootFilesystem:   ptr.To(true),
		RunAsNonRoot:             ptr.To(true),
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		RunAsGroup:               ptr.To(id),
	}
	if !openshift {
		sc.RunAsUser = ptr.To(id)
	}
	return sc
}
