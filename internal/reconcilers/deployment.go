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

// Deployment reconciles a desired *appsv1.Deployment. On update, ResourceVersion is
// preserved from the live object; status and replica count from horizontal scaling are
// left for the API server / autoscaler to manage and re-applied each reconcile.
func Deployment(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService, desired *appsv1.Deployment) error {
	if scheme != nil && cr != nil {
		if err := controllerutil.SetControllerReference(cr, desired, scheme); err != nil {
			return fmt.Errorf("set owner reference: %w", err)
		}
	}
	if err := c.Create(ctx, desired); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("create deployment %s/%s: %w", desired.GetNamespace(), desired.GetName(), err)
		}
		existing := &appsv1.Deployment{}
		if err := c.Get(ctx, client.ObjectKeyFromObject(desired), existing); err != nil {
			return fmt.Errorf("get deployment %s/%s: %w", desired.GetNamespace(), desired.GetName(), err)
		}
		desired.ResourceVersion = existing.ResourceVersion
		return c.Update(ctx, desired)
	}
	return nil
}
