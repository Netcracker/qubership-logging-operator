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

// ServiceAccount reconciles a desired *corev1.ServiceAccount. On update, server-managed
// secrets/imagePullSecrets are preserved when the desired object leaves them empty.
func ServiceAccount(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService, desired *corev1.ServiceAccount) error {
	if scheme != nil && cr != nil {
		if err := controllerutil.SetControllerReference(cr, desired, scheme); err != nil {
			return fmt.Errorf("set owner reference: %w", err)
		}
	}
	if err := c.Create(ctx, desired); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("create service account %s/%s: %w", desired.GetNamespace(), desired.GetName(), err)
		}
		existing := &corev1.ServiceAccount{}
		if err := c.Get(ctx, client.ObjectKeyFromObject(desired), existing); err != nil {
			return fmt.Errorf("get service account %s/%s: %w", desired.GetNamespace(), desired.GetName(), err)
		}
		desired.ResourceVersion = existing.ResourceVersion
		if len(desired.Secrets) == 0 {
			desired.Secrets = existing.Secrets
		}
		if len(desired.ImagePullSecrets) == 0 {
			desired.ImagePullSecrets = existing.ImagePullSecrets
		}
		return c.Update(ctx, desired)
	}
	return nil
}
