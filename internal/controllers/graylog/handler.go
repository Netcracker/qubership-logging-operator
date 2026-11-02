package graylog

import (
	"context"
	"errors"
	"time"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	graylogbuilder "github.com/Netcracker/qubership-logging-operator/internal/builder/graylog"
	"github.com/Netcracker/qubership-logging-operator/internal/reconcilers"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *GraylogReconciler) handleServiceAccount(ctx context.Context, cr *loggingService.LoggingService) error {
	return reconcilers.ServiceAccount(ctx, r.Client, r.Scheme, cr, graylogbuilder.ServiceAccount(cr, r.Defaults))
}

func (r *GraylogReconciler) handleConfigMap(cr *loggingService.LoggingService) error {
	if err := r.setCredentials(cr); err != nil {
		return err
	}
	m, err := graylogConfigMap(cr)
	if err != nil {
		r.Log.Error(err, "Failed creating ConfigMap manifest")
		return err
	}

	if err = r.CreateResource(cr, m); err != nil {
		if api_errors.IsAlreadyExists(err) {
			r.Log.Info("ConfigMap already exists, update it")
			if err = r.UpdateResource(m); err != nil {
				return err
			}
			return nil
		} else {
			return err
		}
	}
	return nil
}

// handleStatefulset applies the StatefulSet and waits until its pods run the new revision.
func (r *GraylogReconciler) handleStatefulset(ctx context.Context, cr *loggingService.LoggingService) error {
	if err := reconcilers.StatefulSet(ctx, r.Client, r.Scheme, cr, graylogbuilder.StatefulSet(cr, r.Defaults)); err != nil {
		return err
	}

	// Delay to allow time for the deploy to be updated
	time.Sleep(util.InitialDelay)

	// Wait for Graylog running
	podManager := util.NewPodManager(r.Client, cr.GetNamespace(), r.Log)
	timeout := util.GraylogStartupTimeout
	if cr.Spec.Graylog.StartupTimeout != 0 {
		timeout = time.Duration(cr.Spec.Graylog.StartupTimeout) * time.Minute
	}
	started, err := podManager.WaitForStatefulsetUpdated(util.GraylogStatefulsetName, timeout)
	if err != nil {
		return err
	}

	if !started {
		r.StatusUpdater.UpdateStatus(util.GraylogStatus, util.Failed, false, "Graylog is not started")
		return errors.New("graylog is not started")
	}
	return nil
}

// handleService applies the Service and waits until Graylog answers on it. Like
// before, the wait runs only when the Service is new.
func (r *GraylogReconciler) handleService(ctx context.Context, cr *loggingService.LoggingService) error {
	service := graylogbuilder.Service(cr, r.Defaults)
	existing := &corev1.Service{}
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(service), existing)
	if err != nil && !api_errors.IsNotFound(err) {
		return err
	}
	created := api_errors.IsNotFound(err)
	if err = reconcilers.Service(ctx, r.Client, r.Scheme, cr, service); err != nil {
		return err
	}
	if !created {
		return nil
	}

	dnsName := util.GraylogComponentName + "." + cr.GetNamespace() + ".svc"
	return util.WaitForHostActive(dnsName, 9000, time.Minute)
}

func (r *GraylogReconciler) deletePVC(name string, cr *loggingService.LoggingService) error {
	e := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	return nil
}

func (r *GraylogReconciler) deleteDeployment(cr *loggingService.LoggingService) error {
	e := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GraylogDeploymentName,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	podManager := util.NewPodManager(r.Client, cr.GetNamespace(), r.Log)
	_, err := podManager.DeletePods(util.GraylogLabels)
	if err != nil {
		return err
	}
	return nil
}

func (r *GraylogReconciler) deleteStatefulset(cr *loggingService.LoggingService) error {
	e := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GraylogStatefulsetName,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	return nil
}

func (r *GraylogReconciler) deleteService(cr *loggingService.LoggingService) error {
	e := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GraylogComponentName,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	return nil
}

func (r *GraylogReconciler) deleteConfigMap(cr *loggingService.LoggingService) error {
	e := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GraylogComponentName,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	return nil
}

func (r *GraylogReconciler) deleteServiceAccount(cr *loggingService.LoggingService) error {
	e := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GraylogServiceAccountName,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	return nil
}

func (r *GraylogReconciler) deleteJob(cr *loggingService.LoggingService, jobName string) error {
	e := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: cr.GetNamespace(),
		},
	}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.DeleteResource(e); err != nil {
		return err
	}
	return nil
}
