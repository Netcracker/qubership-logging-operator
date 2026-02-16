package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PartOfLogging is the value for app.kubernetes.io/part-of on all logging-operator resources.
	PartOfLogging = "logging"
	// ManagedByOperator is the value for app.kubernetes.io/managed-by on resources created by the operator.
	ManagedByOperator = "logging-operator"
)

// CommonLabels returns the labels applied to all resources (part-of, managed-by).
// Single source of truth for operator-created resources; mirrors Helm commonLabels.
func CommonLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/part-of":    PartOfLogging,
		"app.kubernetes.io/managed-by": ManagedByOperator,
	}
}

// ResourceLabels returns name, app.kubernetes.io/name, component, plus CommonLabels.
// Use for any resource (Service, ConfigMap, ServiceAccount, workload, etc.) so labels stay consistent.
func ResourceLabels(name, component string) map[string]string {
	return MergeLabels(
		map[string]string{
			"name":                        name,
			"app.kubernetes.io/name":      name,
			"app.kubernetes.io/component": component,
		},
		CommonLabels(),
	)
}

// MergeLabels returns a new map with all key-value pairs from the given maps.
// Later maps override earlier ones on key conflict. Nil maps are skipped.
func MergeLabels(maps ...map[string]string) map[string]string {
	out := make(map[string]string)
	for _, m := range maps {
		if m == nil {
			continue
		}
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// MergeInto copies all key-value pairs from src into dst. If src is nil, dst is unchanged.
// Caller must ensure dst is non-nil (e.g. after SetLabels or when building a map).
func MergeInto(dst, src map[string]string) {
	if src == nil {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}

// LabelInput holds parameters for setting labels on operator-managed resources.
// Component labels are applied to resource metadata only (not to pod templates).
type LabelInput struct {
	Name            string
	Component       string
	Instance        string
	Version         string
	ComponentLabels map[string]string
}

func (in LabelInput) instanceVersionMap() map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance": in.Instance,
		"app.kubernetes.io/version":  in.Version,
	}
}

// resourceLabelsFromBase returns the full resource label map: base maps merged, then component labels merged in.
func (in LabelInput) resourceLabelsFromBase(baseMaps ...map[string]string) map[string]string {
	out := MergeLabels(baseMaps...)
	MergeInto(out, in.ComponentLabels)
	return out
}

// resourceLabels returns the full resource label map. Existing is merged first; ResourceLabels and instance/version
// are merged after, so standard labels take precedence on key overlap.
func (in LabelInput) resourceLabels(existing map[string]string) map[string]string {
	return in.resourceLabelsFromBase(existing, ResourceLabels(in.Name, in.Component), in.instanceVersionMap())
}

// templateLabels returns the label map for pod template (base only, no component labels).
func (in LabelInput) templateLabels(existing map[string]string) map[string]string {
	return MergeLabels(existing, ResourceLabels(in.Name, in.Component), in.instanceVersionMap())
}

// SetLabelsForResource sets base + component labels on any resource (Service, ServiceAccount, ConfigMap, etc.).
// When existing is nil, the object's current labels (obj.GetLabels()) are used as the initial layer; when non-nil, existing is used instead (e.g. for ConfigMap pass extra labels like k8s-app here, or nil for no initial labels).
func SetLabelsForResource(obj metav1.Object, in LabelInput, existing map[string]string) {
	initial := existing
	if initial == nil {
		initial = obj.GetLabels()
	}
	obj.SetLabels(in.resourceLabels(initial))
}

// SetLabelsForWorkload sets base + component labels on the resource and base-only labels on the pod template.
// Component labels are applied to resource metadata only, not to spec.template.metadata.labels.
// Use for DaemonSet, StatefulSet, Deployment, Job — pass the object and &obj.Spec.Template.Labels.
func SetLabelsForWorkload(obj metav1.Object, templateLabels *map[string]string, in LabelInput) {
	SetLabelsForResource(obj, in, nil)
	*templateLabels = in.templateLabels(*templateLabels)
}
