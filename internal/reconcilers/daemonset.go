package reconcile

import (
	"context"
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// DaemonSet reconciles a desired *appsv1.DaemonSet. On update, ResourceVersion is
// preserved from the live object; status fields are left for the API server to manage.
func DaemonSet(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService, desired *appsv1.DaemonSet) error {
	if scheme != nil && cr != nil {
		if err := controllerutil.SetControllerReference(cr, desired, scheme); err != nil {
			return fmt.Errorf("set owner reference: %w", err)
		}
	}
	if err := c.Create(ctx, desired); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("create daemonset %s/%s: %w", desired.GetNamespace(), desired.GetName(), err)
		}
		existing := &appsv1.DaemonSet{}
		if err := c.Get(ctx, client.ObjectKeyFromObject(desired), existing); err != nil {
			return fmt.Errorf("get daemonset %s/%s: %w", desired.GetNamespace(), desired.GetName(), err)
		}
		desired.ResourceVersion = existing.ResourceVersion
		return c.Update(ctx, desired)
	}
	return nil
}
