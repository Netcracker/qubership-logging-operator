package build

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JobOpts captures the per-Job knobs component factories set.
type JobOpts struct {
	BackoffLimit   *int32
	PodSpec        corev1.PodSpec
	PodLabels      map[string]string
	PodAnnotations map[string]string
	ExtraLabels    map[string]string
	Annotations    map[string]string
}

// NewJob returns a *batchv1.Job with operator-standard labels plus opts. Caller is
// responsible for setting RestartPolicy on PodSpec — batch defaults differ from the
// rest of corev1 and we don't want to override silently.
func NewJob(name, namespace, component string, opts JobOpts) *batchv1.Job {
	meta := ObjectMeta(name, namespace, component)
	for k, v := range opts.ExtraLabels {
		meta.Labels[k] = v
	}
	if len(opts.Annotations) > 0 {
		meta.Annotations = opts.Annotations
	}
	return &batchv1.Job{
		TypeMeta:   metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: meta,
		Spec: batchv1.JobSpec{
			BackoffLimit: opts.BackoffLimit,
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
