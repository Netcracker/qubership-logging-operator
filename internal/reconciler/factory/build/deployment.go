package build

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentOpts captures the per-Deployment knobs used by component factories. Pod-
// level fields (containers, volumes, etc.) come in via PodSpec; the factory composes
// containers via NewContainer first.
type DeploymentOpts struct {
	Replicas       *int32
	Selector       map[string]string
	Strategy       appsv1.DeploymentStrategy
	PodSpec        corev1.PodSpec
	PodLabels      map[string]string
	PodAnnotations map[string]string
	ExtraLabels    map[string]string
	Annotations    map[string]string
}

// NewDeployment returns *appsv1.Deployment with name/namespace/labels from ObjectMeta
// and spec fields from opts. ExtraLabels are merged on top of default resource labels.
func NewDeployment(name, namespace, component string, opts DeploymentOpts) *appsv1.Deployment {
	meta := ObjectMeta(name, namespace, component)
	for k, v := range opts.ExtraLabels {
		meta.Labels[k] = v
	}
	if len(opts.Annotations) > 0 {
		meta.Annotations = opts.Annotations
	}
	return &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: meta,
		Spec: appsv1.DeploymentSpec{
			Replicas: opts.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: opts.Selector},
			Strategy: opts.Strategy,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      opts.PodLabels,
					Annotations: opts.PodAnnotations,
				},
				Spec: opts.PodSpec,
			},
		},
	}
}
