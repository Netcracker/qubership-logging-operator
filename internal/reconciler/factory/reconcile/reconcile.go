// Package reconcile holds generic create-or-update logic per Kubernetes Kind. Each
// per-Kind file (service.go, service_account.go, deployment.go, ...) exposes a single
// entrypoint that takes a fully-built desired object plus the owning LoggingService
// and reconciles it against the cluster.
//
// Stage 0 ships only the generic Apply entrypoint plus Service / ServiceAccount.
// Per-Kind specializations are added as their stages need them.
package reconcile

import (
	"context"
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Object is the union of runtime.Object and metav1.Object that all reconcile helpers
// operate on. Mirrors controllers/utils.K8sResource but lives here so build/ doesn't
// depend on controllers/utils.
type Object interface {
	runtime.Object
	metav1.Object
}

// Apply performs a generic create-or-update for desired against the cluster: it sets a
// controller owner-ref to cr (when scheme is non-nil), tries Create, and on AlreadyExists
// fetches the live object and Updates it after copying name/namespace/owner across. Per-
// Kind helpers (Service, ServiceAccount, Deployment, ...) wrap Apply with kind-specific
// spec-merge logic where blanket Update would clobber server-managed fields.
func Apply(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService, desired Object) error {
	if scheme != nil && cr != nil {
		if err := controllerutil.SetControllerReference(cr, desired, scheme); err != nil {
			return fmt.Errorf("set owner reference: %w", err)
		}
	}
	if err := c.Create(ctx, desired); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("create %s/%s: %w", desired.GetNamespace(), desired.GetName(), err)
		}
		return c.Update(ctx, desired)
	}
	return nil
}
