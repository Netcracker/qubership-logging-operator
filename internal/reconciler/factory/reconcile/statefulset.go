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

// StatefulSet reconciles a desired *appsv1.StatefulSet. On update, ResourceVersion is
// preserved from the live object. The k8s API server rejects mutation of immutable
// fields (serviceName, podManagementPolicy, volumeClaimTemplates, selector); callers
// must keep those stable across reconciles or recreate the resource explicitly.
func StatefulSet(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService, desired *appsv1.StatefulSet) error {
	if scheme != nil && cr != nil {
		if err := controllerutil.SetControllerReference(cr, desired, scheme); err != nil {
			return fmt.Errorf("set owner reference: %w", err)
		}
	}
	if err := c.Create(ctx, desired); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("create statefulset %s/%s: %w", desired.GetNamespace(), desired.GetName(), err)
		}
		existing := &appsv1.StatefulSet{}
		if err := c.Get(ctx, client.ObjectKeyFromObject(desired), existing); err != nil {
			return fmt.Errorf("get statefulset %s/%s: %w", desired.GetNamespace(), desired.GetName(), err)
		}
		desired.ResourceVersion = existing.ResourceVersion
		// Immutable fields: preserve from live to avoid update rejection. If the
		// component spec changed these, the operator must recreate the StatefulSet
		// (out of scope here).
		desired.Spec.ServiceName = existing.Spec.ServiceName
		desired.Spec.PodManagementPolicy = existing.Spec.PodManagementPolicy
		desired.Spec.VolumeClaimTemplates = existing.Spec.VolumeClaimTemplates
		return c.Update(ctx, desired)
	}
	return nil
}
