package builder

import corev1 "k8s.io/api/core/v1"

// First returns a when it is not the zero value, and b otherwise.
func First[T comparable](a, b T) T {
	var zero T
	if a != zero {
		return a
	}
	return b
}

// FirstNonNil returns a when it is not nil, and b otherwise.
func FirstNonNil[T any](a, b *T) *T {
	if a != nil {
		return a
	}
	return b
}

// FirstSlice returns a when it is not empty, and b otherwise.
func FirstSlice[T any](a, b []T) []T {
	if len(a) > 0 {
		return a
	}
	return b
}

// FirstMap returns a when it is not empty, and b otherwise.
func FirstMap[K comparable, V any](a, b map[K]V) map[K]V {
	if len(a) > 0 {
		return a
	}
	return b
}

// Resources returns a copy of the resources set in the custom resource, or of fallback
// when the custom resource sets none. A partial set is used as written: the builder does
// not mix it with fallback values.
func Resources(fromCR *corev1.ResourceRequirements, fallback corev1.ResourceRequirements) corev1.ResourceRequirements {
	if fromCR != nil {
		return *fromCR.DeepCopy()
	}
	return *fallback.DeepCopy()
}
