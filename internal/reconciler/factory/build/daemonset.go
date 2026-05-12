package build

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DaemonSetOpts captures the per-DaemonSet knobs used by component factories. The pod
// spec is composed by the caller (containers via NewContainer, volumes inline) and
// passed in via PodSpec.
type DaemonSetOpts struct {
	Selector          map[string]string
	UpdateStrategy    appsv1.DaemonSetUpdateStrategy
	MinReadySeconds   int32
	PodSpec           corev1.PodSpec
	PodLabels         map[string]string
	PodAnnotations    map[string]string
	PodName           string
	ExtraLabels       map[string]string
	Annotations       map[string]string
}

// NewDaemonSet returns a *appsv1.DaemonSet with the standard operator label set and the
// spec fields from opts. ExtraLabels are merged on top of default resource labels.
func NewDaemonSet(name, namespace, component string, opts DaemonSetOpts) *appsv1.DaemonSet {
	meta := ObjectMeta(name, namespace, component)
	for k, v := range opts.ExtraLabels {
		meta.Labels[k] = v
	}
	if len(opts.Annotations) > 0 {
		meta.Annotations = opts.Annotations
	}
	templateMeta := metav1.ObjectMeta{
		Name:        opts.PodName,
		Labels:      opts.PodLabels,
		Annotations: opts.PodAnnotations,
	}
	return &appsv1.DaemonSet{
		TypeMeta:   metav1.TypeMeta{Kind: "DaemonSet", APIVersion: "apps/v1"},
		ObjectMeta: meta,
		Spec: appsv1.DaemonSetSpec{
			MinReadySeconds: opts.MinReadySeconds,
			Selector:        &metav1.LabelSelector{MatchLabels: opts.Selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: templateMeta,
				Spec:       opts.PodSpec,
			},
			UpdateStrategy: opts.UpdateStrategy,
		},
	}
}
