// Package build holds plain constructors for Kubernetes objects (Container, Service,
// ServiceAccount, Deployment, DaemonSet, StatefulSet, Job) used by per-component
// factories under build/<component>/. These are not fluent builders — each takes an
// option struct and returns the typed object so callers can mutate further if needed.
//
// Coalesce helpers in this file implement the user/CR-wins-over-defaults precedence:
// First(spec.X, cfg.X) returns spec.X when it is non-zero, otherwise cfg.X.
package build

// First returns a if it is non-zero, otherwise b. Use to coalesce a CR-supplied value
// over a code-level default (b).
func First[T comparable](a, b T) T {
	var zero T
	if a != zero {
		return a
	}
	return b
}

// FirstPtr returns *a if a is non-nil, otherwise b. The pointer indirection lets
// callers express "field unset" distinctly from the zero value.
func FirstPtr[T any](a *T, b T) T {
	if a != nil {
		return *a
	}
	return b
}

// FirstSlice returns a if it is non-empty, otherwise b.
func FirstSlice[T any](a, b []T) []T {
	if len(a) > 0 {
		return a
	}
	return b
}

// FirstMap returns a if it is non-empty, otherwise b.
func FirstMap[K comparable, V any](a, b map[K]V) map[K]V {
	if len(a) > 0 {
		return a
	}
	return b
}

// FirstString returns a if it is non-empty, otherwise b. Equivalent to First[string].
func FirstString(a, b string) string { return First(a, b) }
