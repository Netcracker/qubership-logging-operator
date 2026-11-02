// Package reconcilers creates Kubernetes objects built by internal/builder, or updates
// them when they already exist.
//
// An update changes only the fields the operator owns. The rules match what the
// component handlers did before the objects moved from YAML templates to Go:
//
//   - Labels on the object are merged into the existing labels; annotations are left as is.
//   - Workloads get the desired pod template labels and pod spec. Pod template annotations,
//     such as the one `kubectl rollout restart` sets, are kept.
//   - Services get the desired labels, ports, and selector; the allocated cluster IPs stay.
//   - Service accounts get the desired labels.
package reconcilers

import (
	"context"
	"fmt"
	"maps"
	"strings"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// DaemonSet creates the DaemonSet or updates its labels and pod template.
func DaemonSet(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService,
	desired *appsv1.DaemonSet) error {
	return createOrUpdate(ctx, c, scheme, cr, desired, &appsv1.DaemonSet{}, func(existing *appsv1.DaemonSet) {
		existing.SetLabels(mergeLabels(existing.GetLabels(), desired.GetLabels()))
		existing.Spec.Template.SetLabels(desired.Spec.Template.GetLabels())
		existing.Spec.Template.Spec = desired.Spec.Template.Spec
	})
}

// Deployment creates the Deployment or updates its labels, selector, and pod template.
func Deployment(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService,
	desired *appsv1.Deployment) error {
	return createOrUpdate(ctx, c, scheme, cr, desired, &appsv1.Deployment{}, func(existing *appsv1.Deployment) {
		existing.SetLabels(mergeLabels(existing.GetLabels(), desired.GetLabels()))
		existing.Spec.Selector = desired.Spec.Selector
		existing.Spec.Template.SetLabels(desired.Spec.Template.GetLabels())
		existing.Spec.Template.Spec = desired.Spec.Template.Spec
	})
}

// StatefulSet creates the StatefulSet or updates its labels, replicas, and pod template.
// The selector, service name, and volume claim templates are immutable and stay as they are.
func StatefulSet(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService,
	desired *appsv1.StatefulSet) error {
	return createOrUpdate(ctx, c, scheme, cr, desired, &appsv1.StatefulSet{}, func(existing *appsv1.StatefulSet) {
		existing.SetLabels(mergeLabels(existing.GetLabels(), desired.GetLabels()))
		existing.Spec.Replicas = desired.Spec.Replicas
		existing.Spec.Template.SetLabels(desired.Spec.Template.GetLabels())
		existing.Spec.Template.Spec = desired.Spec.Template.Spec
	})
}

// Service creates the Service or updates its labels, ports, and selector.
func Service(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService,
	desired *corev1.Service) error {
	return createOrUpdate(ctx, c, scheme, cr, desired, &corev1.Service{}, func(existing *corev1.Service) {
		existing.SetLabels(desired.GetLabels())
		existing.Spec.Ports = desired.Spec.Ports
		existing.Spec.Selector = desired.Spec.Selector
	})
}

// ServiceAccount creates the ServiceAccount or updates its labels.
func ServiceAccount(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService,
	desired *corev1.ServiceAccount) error {
	return createOrUpdate(ctx, c, scheme, cr, desired, &corev1.ServiceAccount{}, func(existing *corev1.ServiceAccount) {
		existing.SetLabels(desired.GetLabels())
	})
}

// createOrUpdate creates desired with cr as its controller owner. If the object exists,
// it reads the live copy into existing, applies mutate, and writes it back.
func createOrUpdate[T client.Object](ctx context.Context, c client.Client, scheme *runtime.Scheme,
	cr *loggingService.LoggingService, desired T, existing T, mutate func(T)) error {
	if err := setOwner(cr, desired, scheme); err != nil {
		return err
	}
	kind := fmt.Sprintf("%T", desired)
	err := c.Create(ctx, desired)
	if err == nil {
		return nil
	}
	if !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("create %s %s/%s: %w", kind, desired.GetNamespace(), desired.GetName(), err)
	}
	if err = c.Get(ctx, client.ObjectKeyFromObject(desired), existing); err != nil {
		return fmt.Errorf("get %s %s/%s: %w", kind, desired.GetNamespace(), desired.GetName(), err)
	}
	mutate(existing)
	if err = c.Update(ctx, existing); err != nil {
		return fmt.Errorf("update %s %s/%s: %w", kind, desired.GetNamespace(), desired.GetName(), err)
	}
	return nil
}

// setOwner makes cr the controller owner of obj. Like ComponentReconciler.CreateResource,
// it skips the reference when Kubernetes forbids it for a cluster-scoped or
// cross-namespace object.
func setOwner(cr *loggingService.LoggingService, obj client.Object, scheme *runtime.Scheme) error {
	if cr == nil || scheme == nil {
		return nil
	}
	if err := controllerutil.SetControllerReference(cr, obj, scheme); err != nil {
		if strings.Contains(err.Error(), "cluster-scoped resource must not have a namespace-scoped owner") ||
			strings.Contains(err.Error(), "cross-namespace owner references are disallowed") {
			return nil
		}
		return fmt.Errorf("set owner reference on %s/%s: %w", obj.GetNamespace(), obj.GetName(), err)
	}
	return nil
}

func mergeLabels(existing, desired map[string]string) map[string]string {
	if existing == nil {
		return desired
	}
	maps.Copy(existing, desired)
	return existing
}
