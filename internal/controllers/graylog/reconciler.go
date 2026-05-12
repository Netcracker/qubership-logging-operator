package graylog

import (
	"context"
	"errors"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/controllers/graylog/utils"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	graylogfactory "github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build/graylog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GraylogReconciler struct {
	*util.ComponentReconciler
	Cfg *config.Defaults
}

func NewGraylogReconciler(c client.Client, scheme *runtime.Scheme, updater util.StatusUpdater, cfg *config.Defaults) GraylogReconciler {
	return GraylogReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Client:        c,
			Scheme:        scheme,
			Log:           util.Logger("graylog"),
			StatusUpdater: updater,
		},
		Cfg: cfg,
	}
}

// Run reconciles Graylog. ServiceAccount + StatefulSet + Service are built by the Go
// factory; the ConfigMap and MongoDB upgrade orchestration stay in the controller.
func (r *GraylogReconciler) Run(ctx context.Context, cr *loggingService.LoggingService, clientSet kubernetes.Interface) error {
	if !r.StatusUpdater.IsStatusFailed(util.GraylogStatus) {
		r.StatusUpdater.UpdateStatus(util.GraylogStatus, util.InProgress, false, "Start reconcile of Graylog")
	}
	r.Log.Info("Start Graylog reconciliation")

	if cr.Spec.Graylog == nil || !cr.Spec.Graylog.IsInstall() {
		r.Log.Info("Uninstalling component if exists")
		r.uninstall(cr)
		r.StatusUpdater.RemoveStatus(util.GraylogStatus)
		return nil
	}

	connector, err := utils.CreateConnector(ctx, cr, configs, clientSet)
	if err != nil {
		return err
	}
	if err = r.handleConfigMap(cr); err != nil {
		return err
	}
	if err = r.deleteDeployment(cr); err != nil {
		r.Log.Error(err, "Can not delete Deployment")
	}
	if cr.Spec.Graylog.MongoDBUpgrade != nil && r.checkGraylog5(cr) {
		if err = r.mongoUpgrade(cr); err != nil {
			r.Log.Error(err, "MongoDB upgrade failed. Try to continue the reconciliation anyway")
		}
	} else {
		if err = r.deleteUpgradeJobs(cr); err != nil {
			r.Log.Error(err, "Can not delete MongoDB upgrade jobs")
		}
	}
	_, err = util.ReconcileAndTrackStatus(ctx, r.Client, cr, &r.StatusUpdater, util.GraylogStatus, func() (ctrl.Result, error) {
		return ctrl.Result{}, graylogfactory.CreateOrUpdate(ctx, r.Client, r.Scheme, cr, r.Cfg)
	})
	if err != nil {
		return err
	}
	if err = r.waitForGraylogReady(cr); err != nil {
		return err
	}
	if err = r.waitForServiceReachable(cr); err != nil {
		return err
	}

	if cr.Spec.Graylog.ContentDeployPolicy != "skip" {
		if err = r.configureGraylog(ctx, connector, cr, clientSet); err != nil {
			return err
		}
	}
	if err = r.watchSecret(cr); err != nil {
		r.Log.Error(err, "Error occurred while starting watch Secret")
	}

	r.Log.Info("Component reconciled")
	r.StatusUpdater.RemoveStatus(util.GraylogStatus)
	return nil
}

// uninstall deletes all resources related to the component.
func (r *GraylogReconciler) uninstall(cr *loggingService.LoggingService) {
	if err := r.deletePVC(util.GraylogClaimName, cr); err != nil {
		r.Log.Error(err, "Can not delete graylog PVC")
	}
	if err := r.deletePVC(util.MongoClaimName, cr); err != nil {
		r.Log.Error(err, "Can not delete mongo PVC")
	}
	if err := r.deleteStatefulset(cr); err != nil {
		r.Log.Error(err, "Can not delete Statefulset")
	}
	if err := r.deleteService(cr); err != nil {
		r.Log.Error(err, "Can not delete Service")
	}
	if err := r.deleteConfigMap(cr); err != nil {
		r.Log.Error(err, "Can not delete ConfigMap")
	}
	if err := r.deleteServiceAccount(cr); err != nil {
		r.Log.Error(err, "Can not delete ServiceAccount")
	}
	if err := r.deleteUpgradeJobs(cr); err != nil {
		r.Log.Error(err, "Can not delete MongoDB upgrade jobs")
	}
}

func (r *GraylogReconciler) configureGraylog(ctx context.Context, connector *utils.GraylogConnector, cr *loggingService.LoggingService, clientSet kubernetes.Interface) error {
	if cr.Spec.Graylog.AuthProxy.Install {
		if err := connector.ManageAuthHeaderConfig(cr); err != nil {
			return err
		}
	}
	if err := connector.ManageGrokPatterns(cr); err != nil {
		return err
	}
	if err := connector.ManageIndexSets(cr); err != nil {
		return err
	}
	if err := connector.ManageInputs(cr); err != nil {
		return err
	}
	if err := connector.ManageExtractors(cr, r.checkGraylog5(cr)); err != nil {
		return err
	}
	if err := connector.ManageStreams(cr); err != nil {
		return err
	}
	if err := connector.ManageProcessingRules(cr); err != nil {
		return err
	}
	if err := connector.ManagePipelines(cr); err != nil {
		return err
	}
	if err := connector.ManageDashboards(cr); err != nil {
		return err
	}
	if cr.Spec.Graylog.ContentPackPaths != "" {
		if err := connector.ManageContentPacks(cr); err != nil {
			return err
		}
		if err := connector.ManageOpensearchConfigs(cr); err != nil {
			return err
		}
	}
	if cr.Spec.Graylog.ContentPacks != nil {
		if err := connector.ManageContentPackTLS(ctx, cr, clientSet); err != nil {
			return err
		}
		if err := connector.ManageOpensearchConfigs(cr); err != nil {
			return err
		}
	}
	if err := connector.ManageArchivesDirectory(cr); err != nil {
		return err
	}
	if err := connector.ManageSavedSearches(cr); err != nil {
		return err
	}
	if err := connector.ManageUserAccounts(cr, r.checkGraylog5(cr)); err != nil {
		return err
	}
	return nil
}

func (r *GraylogReconciler) setCredentials(cr *loggingService.LoggingService) error {
	secret := &corev1.Secret{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name: cr.Spec.Graylog.GraylogSecretName, Namespace: cr.GetNamespace(),
	}, secret); err != nil {
		return err
	}

	if secret.Data != nil && len(secret.Data["user"]) > 0 {
		cr.Spec.Graylog.User = string(secret.Data["user"])
	} else {
		return errors.New("can not find user for Graylog in the secret " + cr.Spec.Graylog.GraylogSecretName + " in the namespace " + cr.GetNamespace())
	}
	if secret.Data != nil && len(secret.Data["password"]) > 0 {
		cr.Spec.Graylog.Password = string(secret.Data["password"])
	} else {
		return errors.New("can not find password for Graylog in the secret " + cr.Spec.Graylog.GraylogSecretName + " in the namespace " + cr.GetNamespace())
	}
	return nil
}

func (r *GraylogReconciler) watchSecret(cr *loggingService.LoggingService) error {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return err
	}
	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	w := utils.SecretEventWatcher{
		Log:       util.Logger("watch-secret"),
		Clientset: k8sClient,
	}
	go w.Watch(cr, r.Client)
	return nil
}

func (r *GraylogReconciler) mongoUpgrade(cr *loggingService.LoggingService) error {
	if err := r.scaleDownStatefulset(cr); err != nil {
		return err
	}
	for _, jobName := range util.GraylogMongoUpgradeOrderedJobs {
		if err := r.handleMongoUpgradeJob(cr, graylogfactory.UpgradeJobName(jobName)); err != nil {
			return err
		}
	}
	return nil
}

func (r *GraylogReconciler) deleteUpgradeJobs(cr *loggingService.LoggingService) error {
	for _, jobName := range util.GraylogMongoUpgradeOrderedJobs {
		if err := r.deleteJob(cr, jobName); err != nil {
			return err
		}
	}
	return nil
}

// checkGraylog5 delegates to the factory's version detector so the controller and
// factory share one implementation. Tested by TestCheckGraylog5.
func (r *GraylogReconciler) checkGraylog5(cr *loggingService.LoggingService) bool {
	return graylogfactory.IsV5(cr.Spec.Graylog.DockerImage)
}
