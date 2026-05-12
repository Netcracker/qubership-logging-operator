// Package fluentbit builds the FluentBit DaemonSet and Service from the LoggingService
// CR plus code-level defaults. It replaces the embedded YAML template at
// controllers/fluentbit/assets/. The ConfigMap path is intentionally left in place in
// controllers/fluentbit/manifest.go; this package handles only Kubernetes deployment
// objects.
package fluentbit

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
	// ComponentName matches util.FluentbitComponentName and is used for both the
	// app.kubernetes.io/component label and the DaemonSet / Service / config-volume
	// configmap name.
	ComponentName = "logging-fluentbit"
	// MainContainer is the name of the FluentBit container inside the DaemonSet pod.
	MainContainer = "logging-fluentbit"
	// ReloadContainer is the configmap-reload sidecar container name.
	ReloadContainer = "configmap-reload"
	// Technology is the value of app.kubernetes.io/technology on the pod template.
	Technology = "c"
)

// CreateOrUpdate reconciles the FluentBit DaemonSet and Service. Caller gates on
// cr.Spec.Fluentbit being non-nil and aggregator HA mode being disabled.
func CreateOrUpdate(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService, cfg *config.Defaults) error {
	ds := buildDaemonSet(cr, cfg.Fluentbit)
	if err := reconcile.DaemonSet(ctx, c, scheme, cr, ds); err != nil {
		return err
	}
	return reconcile.Service(ctx, c, scheme, cr, buildService(cr, cfg.Fluentbit))
}

// selector returns the pod-template label set used to wire Service → pods and
// DaemonSet → pods. Mirrors the asset's selector exactly.
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

// podLabels are the labels set on spec.template.metadata. Includes the cluster-service
// marker historically baked into the asset.
func podLabels(openshift bool) map[string]string {
	l := selector(openshift)
	l["app.kubernetes.io/technology"] = Technology
	l["kubernetes.io/cluster-service"] = "true"
	return l
}

// resourceLabels are the labels set on the DaemonSet and Service metadata. They mirror
// the workload pod labels plus the operator-standard part-of / managed-by / etc.
// applied later via util.SetLabelsForWorkload / util.SetLabelsForResource.
func resourceLabels(openshift bool) map[string]string {
	l := podLabels(openshift)
	l["app.kubernetes.io/technology"] = Technology
	return l
}

func buildService(cr *loggingService.LoggingService, def config.FluentbitDefaults) *corev1.Service {
	spec := cr.Spec.Fluentbit
	httpPort := def.HTTPPort
	metricsPort := def.LogToMetricsPort
	svc := build.NewService(ComponentName, cr.GetNamespace(), ComponentName, build.ServiceOpts{
		Type: corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{
			{Name: ComponentName, Port: httpPort, TargetPort: intstr.FromInt32(httpPort), Protocol: corev1.ProtocolTCP},
			{Name: "log-to-metrics", Port: metricsPort, TargetPort: intstr.FromInt32(metricsPort), Protocol: corev1.ProtocolTCP},
		},
		Selector:    map[string]string{"name": ComponentName},
		ExtraLabels: serviceProviderLabels(cr.Spec.OpenshiftDeploy),
	})
	util.SetLabelsForResource(svc, util.LabelInput{
		Name:            ComponentName,
		Component:       "fluentbit",
		ComponentLabels: spec.Labels,
	}, nil)
	return svc
}

// serviceProviderLabels matches the legacy asset's Service-only label set:
// component=logging-fluentbit + provider=<kubernetes|openshift>. The component label
// is added by util.ResourceLabels under "app.kubernetes.io/component" but the asset
// also wrote a bare "component" key — preserved for compatibility.
func serviceProviderLabels(openshift bool) map[string]string {
	provider := "kubernetes"
	if openshift {
		provider = "openshift"
	}
	return map[string]string{
		"component": ComponentName,
		"provider":  provider,
	}
}

func buildDaemonSet(cr *loggingService.LoggingService, def config.FluentbitDefaults) *appsv1.DaemonSet {
	spec := cr.Spec.Fluentbit
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
	ds := build.NewDaemonSet(ComponentName, cr.GetNamespace(), "fluentbit", build.DaemonSetOpts{
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
		ExtraLabels:    resourceLabels(openshift),
		Annotations:    spec.Annotations,
	})

	util.SetLabelsForWorkload(ds, &ds.Spec.Template.Labels, util.LabelInput{
		Name:            ComponentName,
		Component:       "fluentbit",
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
