package build

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ObjectMeta returns a metav1.ObjectMeta populated with name, namespace, and the
// resource label set produced by util.ResourceLabels. Component-specific extra labels
// can be merged via util.MergeLabels by the caller before assignment.
func ObjectMeta(name, namespace, component string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    util.ResourceLabels(name, component),
	}
}

// SetOwner sets a controller reference from the given LoggingService to obj. Errors
// from cross-namespace owner-ref attempts are passed through; callers should propagate
// them up the reconciler stack.
func SetOwner(cr *loggingService.LoggingService, obj metav1.Object, scheme *runtime.Scheme) error {
	return controllerutil.SetControllerReference(cr, obj, scheme)
}
