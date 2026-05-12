package graylog

import (
	"context"
	"errors"
	"time"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	graylogfactory "github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build/graylog"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/reconcile"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// handleConfigMap reconciles the Graylog ConfigMap. ServiceAccount / StatefulSet /
// Service moved to the Go factory in Stage 5.
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
			return r.UpdateResource(m)
		}
		return err
	}
	return nil
}

// handleMongoUpgradeJob applies one MongoDB feature-compatibility upgrade Job and
// waits for it to finish. The Job builder lives in the factory; orchestration (delay,
// wait-for-success, delete-pods, fail-fast) stays here to keep the factory pure.
func (r *GraylogReconciler) handleMongoUpgradeJob(cr *loggingService.LoggingService, name graylogfactory.UpgradeJobName) error {
	job := graylogfactory.BuildMongoUpgradeJob(cr, name, r.Cfg)
	if err := reconcile.Job(context.TODO(), r.Client, r.Scheme, cr, job); err != nil {
		r.Log.Error(err, "Failed creating Job for MongoDB upgrade")
		return err
	}

	time.Sleep(util.InitialDelay)
	podManager := util.NewPodManager(r.Client, cr.GetNamespace(), r.Log)
	succeeded, err := podManager.WaitForJobSucceeded(string(name), util.GraylogMongoUpgradeJobTimeout)
	if err != nil {
		return err
	}
	if !succeeded {
		r.StatusUpdater.UpdateStatus(util.GraylogStatus, util.Failed, false, "Job failed")
		return errors.New("mongo upgrade job failed")
	}
	if _, err = podManager.DeletePods(util.GraylogMongoUpgradeLabels); err != nil {
		return err
	}
	return nil
}

// waitForGraylogReady is the post-apply readiness wait the legacy handleStatefulset
// performed. The factory does the apply; this routine watches the rollout.
func (r *GraylogReconciler) waitForGraylogReady(cr *loggingService.LoggingService) error {
	time.Sleep(util.InitialDelay)
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

// waitForServiceReachable performs the legacy handleService' post-apply DNS+TCP wait.
func (r *GraylogReconciler) waitForServiceReachable(cr *loggingService.LoggingService) error {
	dnsName := util.GraylogComponentName + "." + cr.GetNamespace() + ".svc"
	return util.WaitForHostActive(dnsName, 9000, time.Minute)
}

func (r *GraylogReconciler) deletePVC(name string, cr *loggingService.LoggingService) error {
	e := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cr.GetNamespace()}}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

func (r *GraylogReconciler) deleteDeployment(cr *loggingService.LoggingService) error {
	e := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: util.GraylogDeploymentName, Namespace: cr.GetNamespace()}}
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
	return err
}

func (r *GraylogReconciler) deleteStatefulset(cr *loggingService.LoggingService) error {
	e := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: util.GraylogStatefulsetName, Namespace: cr.GetNamespace()}}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

func (r *GraylogReconciler) deleteService(cr *loggingService.LoggingService) error {
	e := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: util.GraylogComponentName, Namespace: cr.GetNamespace()}}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

func (r *GraylogReconciler) deleteConfigMap(cr *loggingService.LoggingService) error {
	e := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: util.GraylogComponentName, Namespace: cr.GetNamespace()}}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

func (r *GraylogReconciler) deleteServiceAccount(cr *loggingService.LoggingService) error {
	e := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: util.GraylogServiceAccountName, Namespace: cr.GetNamespace()}}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

func (r *GraylogReconciler) deleteJob(cr *loggingService.LoggingService, jobName string) error {
	e := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: cr.GetNamespace()}}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.DeleteResource(e)
}

func (r *GraylogReconciler) scaleDownStatefulset(cr *loggingService.LoggingService) error {
	e := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: util.GraylogStatefulsetName, Namespace: cr.GetNamespace()}}
	if err := r.GetResource(e); err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	zero := int32(0)
	e.Spec.Replicas = &zero
	if err := r.UpdateResource(e); err != nil {
		return err
	}
	time.Sleep(util.InitialDelay)
	podManager := util.NewPodManager(r.Client, cr.GetNamespace(), r.Log)
	timeout := util.GraylogStartupTimeout
	if cr.Spec.Graylog.StartupTimeout != 0 {
		timeout = time.Duration(cr.Spec.Graylog.StartupTimeout) * time.Minute
	}
	updated, err := podManager.WaitForStatefulsetUpdated(util.GraylogStatefulsetName, timeout)
	if err != nil {
		return err
	}
	if !updated {
		r.StatusUpdater.UpdateStatus(util.GraylogStatus, util.Failed, false, "Graylog has not scaled down")
		return errors.New("graylog has not scaled down")
	}
	return nil
}
