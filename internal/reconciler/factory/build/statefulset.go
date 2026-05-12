package build

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatefulSetOpts captures the per-StatefulSet knobs component factories set. The pod
// spec is composed by the caller and passed in via PodSpec; volume claim templates are
// optional (HA aggregator uses emptyDir by default and a PVC only when binding).
type StatefulSetOpts struct {
	Replicas             *int32
	ServiceName          string
	Selector             map[string]string
	UpdateStrategy       appsv1.StatefulSetUpdateStrategy
	PodSpec              corev1.PodSpec
	PodLabels            map[string]string
	PodAnnotations       map[string]string
	PodName              string
	VolumeClaimTemplates []corev1.PersistentVolumeClaim
	RevisionHistoryLimit *int32
	ExtraLabels          map[string]string
	Annotations          map[string]string
}

// NewStatefulSet returns a *appsv1.StatefulSet. ExtraLabels are merged on top of the
// default resource labels produced by ObjectMeta.
func NewStatefulSet(name, namespace, component string, opts StatefulSetOpts) *appsv1.StatefulSet {
	meta := ObjectMeta(name, namespace, component)
	for k, v := range opts.ExtraLabels {
		meta.Labels[k] = v
	}
	if len(opts.Annotations) > 0 {
		meta.Annotations = opts.Annotations
	}
	return &appsv1.StatefulSet{
		TypeMeta:   metav1.TypeMeta{Kind: "StatefulSet", APIVersion: "apps/v1"},
		ObjectMeta: meta,
		Spec: appsv1.StatefulSetSpec{
			Replicas:    opts.Replicas,
			ServiceName: opts.ServiceName,
			Selector:    &metav1.LabelSelector{MatchLabels: opts.Selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        opts.PodName,
					Labels:      opts.PodLabels,
					Annotations: opts.PodAnnotations,
				},
				Spec: opts.PodSpec,
			},
			UpdateStrategy:       opts.UpdateStrategy,
			VolumeClaimTemplates: opts.VolumeClaimTemplates,
			RevisionHistoryLimit: opts.RevisionHistoryLimit,
		},
	}
}
