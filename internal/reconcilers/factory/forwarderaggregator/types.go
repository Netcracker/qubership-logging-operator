package forwarderaggregator

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// metav1ObjectMeta is a local alias to keep the aggregator file readable.
type metav1ObjectMeta = metav1.ObjectMeta

// stringPtrOrNil returns &s if non-empty, else nil. Useful for optional API fields
// where empty-string is semantically different from "not set" (storageClassName).
func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
