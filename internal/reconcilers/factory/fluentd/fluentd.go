// Package fluentd builds the FluentD DaemonSet and Service from the LoggingService CR
// plus code-level defaults. It replaces the embedded YAML template at
// controllers/fluentd/assets/. The ConfigMap path stays in controllers/fluentd/manifest.go.
package fluentd

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
	// ComponentName matches util.FluentdComponentName.
	ComponentName = "logging-fluentd"
	// MainContainer is the FluentD container name.
	MainContainer = "logging-fluentd"
	// ReloadContainer is the configmap-reload sidecar.
	ReloadContainer = "configmap-reload"
	// Technology is set on the pod-template app.kubernetes.io/technology label.
	Technology = "ruby"
)

// CreateOrUpdate reconciles the FluentD DaemonSet and Service. Caller gates on
// cr.Spec.Fluentd.IsInstall().
func CreateOrUpdate(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService, cfg *config.Defaults) error {
	ds := buildDaemonSet(cr, cfg.Fluentd)
	if err := reconcile.DaemonSet(ctx, c, scheme, cr, ds); err != nil {
		return err
	}
	return reconcile.Service(ctx, c, scheme, cr, buildService(cr, cfg.Fluentd))
}

func selector(openshift bool) map[string]string {
	provider := "kubernetes"
	if openshift {
		provider = "openshift"
	}
	return map[string]string{
		"component": ComponentName,
		"provider":  provider,
	}
}

func podLabels(openshift bool) map[string]string {
	l := selector(openshift)
	l["app.kubernetes.io/technology"] = Technology
	return l
}

// providerLabels mirrors the legacy asset's resource label set (bare "component" and
// "provider" keys, in addition to the operator-standard labels added later by
// util.SetLabels*).
func providerLabels(openshift bool) map[string]string {
	provider := "kubernetes"
	if openshift {
		provider = "openshift"
	}
	return map[string]string{
		"component": ComponentName,
		"provider":  provider,
	}
}

func buildService(cr *loggingService.LoggingService, def config.FluentdDefaults) *corev1.Service {
	spec := cr.Spec.Fluentd
	port := def.HTTPPort
	svc := build.NewService(ComponentName, cr.GetNamespace(), ComponentName, build.ServiceOpts{
		Type: corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{
			{Name: ComponentName, Port: port, TargetPort: intstr.FromInt32(port), Protocol: corev1.ProtocolTCP},
		},
		Selector:    map[string]string{"name": ComponentName},
		ExtraLabels: providerLabels(cr.Spec.OpenshiftDeploy),
	})
	util.SetLabelsForResource(svc, util.LabelInput{
		Name:            ComponentName,
		Component:       "fluentd",
		ComponentLabels: spec.Labels,
	}, nil)
	return svc
}

func buildDaemonSet(cr *loggingService.LoggingService, def config.FluentdDefaults) *appsv1.DaemonSet {
	spec := cr.Spec.Fluentd
	openshift := cr.Spec.OpenshiftDeploy

	containers := []corev1.Container{
		buildConfigmapReloadContainer(spec, def),
		buildMainContainer(cr, def),
	}
	volumes := buildVolumes(cr)
	terminationGrace := def.TerminationGracePeriodSeconds

	pod := corev1.PodSpec{
		TerminationGracePeriodSeconds: &terminationGrace,
		ServiceAccountName:            ComponentName,
		Tolerations:                   build.FirstSlice(spec.Tolerations, def.Tolerations),
		NodeSelector:                  nodeSelector(spec.NodeSelectorKey, spec.NodeSelectorValue),
		Affinity:                      spec.Affinity,
		PriorityClassName:             spec.PriorityClassName,
		Volumes:                       volumes,
		Containers:                    containers,
	}

	maxUnavailable := def.MaxUnavailable
	ds := build.NewDaemonSet(ComponentName, cr.GetNamespace(), "fluentd", build.DaemonSetOpts{
		Selector:        selector(openshift),
		MinReadySeconds: def.MinReadySeconds,
		UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
			Type: appsv1.RollingUpdateDaemonSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: &maxUnavailable,
			},
		},
		PodSpec:        pod,
		PodName:        ComponentName,
		PodLabels:      podLabels(openshift),
		PodAnnotations: spec.Annotations,
		ExtraLabels:    providerLabels(openshift),
		Annotations:    spec.Annotations,
	})

	util.SetLabelsForWorkload(ds, &ds.Spec.Template.Labels, util.LabelInput{
		Name:            ComponentName,
		Component:       "fluentd",
		Instance:        util.GetInstanceLabel(ComponentName, cr.GetNamespace()),
		Version:         util.GetTagFromImage(spec.DockerImage),
		Technology:      Technology,
		ComponentLabels: spec.Labels,
	})
	return ds
}

func nodeSelector(key, value string) map[string]string {
	if key == "" || value == "" {
		return nil
	}
	return map[string]string{key: value}
}
