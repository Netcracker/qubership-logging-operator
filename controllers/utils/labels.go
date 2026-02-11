package utils

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
