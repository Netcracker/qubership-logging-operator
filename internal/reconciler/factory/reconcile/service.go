package reconcile

import (
	"context"
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Service reconciles a desired *corev1.Service. On update, it preserves server-assigned
// ClusterIP / ClusterIPs / IPFamilies / ResourceVersion to avoid spurious "field is
// immutable" errors that a blanket Update would trigger.
func Service(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService, desired *corev1.Service) error {
	if scheme != nil && cr != nil {
		if err := controllerutil.SetControllerReference(cr, desired, scheme); err != nil {
			return fmt.Errorf("set owner reference: %w", err)
		}
	}
	if err := c.Create(ctx, desired); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("create service %s/%s: %w", desired.GetNamespace(), desired.GetName(), err)
		}
		existing := &corev1.Service{}
		if err := c.Get(ctx, client.ObjectKeyFromObject(desired), existing); err != nil {
			return fmt.Errorf("get service %s/%s: %w", desired.GetNamespace(), desired.GetName(), err)
		}
		desired.ResourceVersion = existing.ResourceVersion
		desired.Spec.ClusterIP = existing.Spec.ClusterIP
		desired.Spec.ClusterIPs = existing.Spec.ClusterIPs
		if len(desired.Spec.IPFamilies) == 0 {
			desired.Spec.IPFamilies = existing.Spec.IPFamilies
		}
		if desired.Spec.IPFamilyPolicy == nil {
			desired.Spec.IPFamilyPolicy = existing.Spec.IPFamilyPolicy
		}
		return c.Update(ctx, desired)
	}
	return nil
}
