package build

import (
	corev1 "k8s.io/api/core/v1"
)

// ServiceOpts captures the fields component factories typically set on a Service.
// Selector is the pod-template label set used to select endpoints; it should match the
// owning workload's spec.template.metadata.labels.
type ServiceOpts struct {
	Type        corev1.ServiceType
	Ports       []corev1.ServicePort
	Selector    map[string]string
	ClusterIP   string
	IPFamilies  []corev1.IPFamily
	IPFamilyPol *corev1.IPFamilyPolicy
	Annotations map[string]string
	ExtraLabels map[string]string
}

// NewService returns a *corev1.Service with name/namespace/labels from ObjectMeta and
// spec fields from opts. ExtraLabels are merged on top of the default resource labels.
func NewService(name, namespace, component string, opts ServiceOpts) *corev1.Service {
	meta := ObjectMeta(name, namespace, component)
	if len(opts.ExtraLabels) > 0 {
		for k, v := range opts.ExtraLabels {
			meta.Labels[k] = v
		}
	}
	if len(opts.Annotations) > 0 {
		meta.Annotations = opts.Annotations
	}
	return &corev1.Service{
		TypeMeta: serviceTypeMeta(),
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Type:           opts.Type,
			Ports:          opts.Ports,
			Selector:       opts.Selector,
			ClusterIP:      opts.ClusterIP,
			IPFamilies:     opts.IPFamilies,
			IPFamilyPolicy: opts.IPFamilyPol,
		},
	}
}
