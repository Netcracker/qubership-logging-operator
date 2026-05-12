package reconcile

import (
	"context"
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Job creates the desired *batchv1.Job. If the Job already exists the call is a no-op
// — almost every field of a Job is immutable after creation, and the historical
// behavior was to skip-if-exists rather than recreate, leaving cleanup to the
// caller (e.g. deleteUpgradeJobs).
func Job(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService, desired *batchv1.Job) error {
	if scheme != nil && cr != nil {
		if err := controllerutil.SetControllerReference(cr, desired, scheme); err != nil {
			return fmt.Errorf("set owner reference: %w", err)
		}
	}
	if err := c.Create(ctx, desired); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("create job %s/%s: %w", desired.GetNamespace(), desired.GetName(), err)
	}
	return nil
}
