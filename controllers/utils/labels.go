package utils

import (
	"fmt"
	"maps"
	maps0 "maps"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PartOfLogging is the value for app.kubernetes.io/part-of on all logging-operator resources.
	PartOfLogging = "logging"
	// ManagedByOperator is the value for app.kubernetes.io/managed-by on resources created by the operator.
	ManagedByOperator = "logging-operator"
	// OperatorDeploymentName is the name of the logging-operator Deployment; used for app.kubernetes.io/managed-by-operator.
	OperatorDeploymentName = "logging-service-operator"
)

// CommonLabels returns the labels applied to all resources (part-of, managed-by, managed-by-operator).
// Single source of truth for operator-created resources; mirrors Helm commonLabels.
func CommonLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/part-of":             PartOfLogging,
		"app.kubernetes.io/managed-by":          ManagedByOperator,
		"app.kubernetes.io/managed-by-operator": OperatorDeploymentName,
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
		maps0.Copy(out, m)
	}
	return out
}

// MergeInto copies all key-value pairs from src into dst. If src is nil, dst is unchanged.
// Caller must ensure dst is non-nil (e.g. after SetLabels or when building a map).
func MergeInto(dst, src map[string]string) {
	if src == nil {
		return
	}
	maps.Copy(dst, src)
}

// TruncLabel truncates a label value to 63 characters (Kubernetes limit). Use for name, app.kubernetes.io/name.
func TruncLabel(label string) string {
	if len(label) >= 63 {
		return strings.Trim(label[:63], "-")
	}
	return strings.Trim(label, "-")
}

// GetInstanceLabel returns a truncated label value for app.kubernetes.io/instance (name-namespace).
func GetInstanceLabel(name, namespace string) string {
	return TruncLabel(fmt.Sprintf("%s-%s", name, namespace))
}

// LabelInput holds parameters for setting labels on operator-managed resources.
// ComponentLabels are applied to resource metadata only (not to pod templates) and override base labels on key conflict.
type LabelInput struct {
	Name            string
	Component       string
	Instance        string
	Version         string
	Technology      string
	ComponentLabels map[string]string
}

// BaseOnlyLabelInput returns LabelInput with base labels only (no instance, version, technology).
// Use for ServiceAccount, ClusterRole, Service, ServiceMonitor, etc. per label specification.
func BaseOnlyLabelInput(name, component string) LabelInput {
	return LabelInput{Name: name, Component: component}
}

func (in LabelInput) instanceVersionTechnologyMap() map[string]string {
	m := make(map[string]string)
	if in.Instance != "" {
		m["app.kubernetes.io/instance"] = in.Instance
	}
	if in.Version != "" {
		m["app.kubernetes.io/version"] = in.Version
	}
	if in.Technology != "" {
		m["app.kubernetes.io/technology"] = in.Technology
	}
	return m
}

// resourceLabelsFromBase returns the full resource label map: base maps merged, then component labels merged in.
func (in LabelInput) resourceLabelsFromBase(baseMaps ...map[string]string) map[string]string {
	out := MergeLabels(baseMaps...)
	MergeInto(out, in.ComponentLabels)
	return out
}

// resourceLabels returns the full resource label map. Merge order: existing, ResourceLabels, instance/version/technology, ComponentLabels.
// ComponentLabels override earlier layers on key conflict.
func (in LabelInput) resourceLabels(existing map[string]string) map[string]string {
	return in.resourceLabelsFromBase(existing, ResourceLabels(in.Name, in.Component), in.instanceVersionTechnologyMap())
}

// templateLabels returns the label map for pod template (base + instance/version/technology; no ComponentLabels).
func (in LabelInput) templateLabels(existing map[string]string) map[string]string {
	return MergeLabels(existing, ResourceLabels(in.Name, in.Component), in.instanceVersionTechnologyMap())
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

// SetLabelsForWorkload sets base + component labels on the resource and base + instance/version/technology on the pod template.
// ComponentLabels are applied to the resource only, not to spec.template.metadata.labels.
// Use for DaemonSet, StatefulSet, Deployment, Job — pass the object and &obj.Spec.Template.Labels.
func SetLabelsForWorkload(obj metav1.Object, templateLabels *map[string]string, in LabelInput) {
	SetLabelsForResource(obj, in, nil)
	*templateLabels = in.templateLabels(*templateLabels)
}

// PodTemplateLabels returns the label map for pod template (base + instance + version + technology).
// Use for CRs with PodMetadata or other pod-label needs.
func PodTemplateLabels(name, component, instance, version, technology string) map[string]string {
	in := LabelInput{Name: name, Component: component, Instance: instance, Version: version, Technology: technology}
	return in.templateLabels(nil)
}
