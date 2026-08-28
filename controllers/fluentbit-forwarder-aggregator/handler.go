package fluentbit_forwarder_aggregator

import (
	"errors"
	"fmt"
	"maps"
	"time"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *HAFluentReconciler) handleForwarderConfigMap(cr *loggingService.LoggingService) error {
	m, err := forwarderConfigMap(cr, r.DynamicParameters)
	if err != nil {
		r.Log.Error(err, "Failed creating ConfigMap manifest")
		return err
	}

	_, err = r.updateConfigMap(cr, m)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Cannot create or update config map %s", m.Name))
		return err
	}

	return nil
}

func (r *HAFluentReconciler) handleForwarderDaemonSet(cr *loggingService.LoggingService) error {
	m, err := forwarderDaemonSet(cr, r.DynamicParameters)
	if err != nil {
		r.Log.Error(err, "Failed creating DaemonSet manifest")
		return err
	}

	err = r.CreateResource(cr, m)
	if err == nil {
		return nil
	}
	if !api_errors.IsAlreadyExists(err) {
		return err
	}

	return r.updateForwarderDaemonSet(m)
}

func (r *HAFluentReconciler) updateForwarderDaemonSet(desired *appsv1.DaemonSet) error {
	existing := &appsv1.DaemonSet{ObjectMeta: desired.ObjectMeta}
	if err := r.GetResource(existing); err != nil {
		return err
	}

	if existing.Labels == nil && desired.Labels != nil {
		existing.SetLabels(desired.Labels)
	} else {
		maps.Copy(existing.Labels, desired.Labels)
	}
	existing.Spec.Template.SetLabels(desired.Spec.Template.GetLabels())
	existing.Spec.Template.Spec = desired.Spec.Template.Spec

	return r.UpdateResource(existing)
}

func (r *HAFluentReconciler) handleForwarderService(cr *loggingService.LoggingService) error {
	m, err := forwarderService(cr, r.DynamicParameters)
	if err != nil {
		r.Log.Error(err, "Failed creating Service manifest")
		return err
	}

	if err = r.CreateResource(cr, m); err != nil {
		if api_errors.IsAlreadyExists(err) {
			e := &corev1.Service{ObjectMeta: m.ObjectMeta}
			if err = r.GetResource(e); err != nil {
				return err
			}

			//Set parameters
			e.SetLabels(m.GetLabels())
			e.Spec.Ports = m.Spec.Ports
			e.Spec.Selector = m.Spec.Selector

			if err = r.UpdateResource(e); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func (r *HAFluentReconciler) Equal(source, target *corev1.ConfigMap) bool {
	return cmp.Equal(source.Data, target.Data) &&
		cmp.Equal(source.BinaryData, target.BinaryData) &&
		cmp.Equal(source.GetLabels(), target.GetLabels())
}

func (r *HAFluentReconciler) CreateOrUpdate(cr *loggingService.LoggingService, configMap *corev1.ConfigMap) (created bool, updated bool, err error) {
	if err = r.CreateResource(cr, configMap); err != nil {
		if api_errors.IsAlreadyExists(err) {
			existedConfigMap := &corev1.ConfigMap{ObjectMeta: configMap.ObjectMeta}
			if err = r.GetResource(existedConfigMap); err != nil {
				return false, false, err
			}

			if !r.Equal(existedConfigMap, configMap) {
				if err = r.UpdateResource(configMap); err != nil {
					return false, false, err
				}

				return false, true, nil
			}

			r.Log.Info("The config map is not changed")
			return false, false, nil
		}

		return false, false, err
	}

	return true, false, nil
}

func (r *HAFluentReconciler) handleAggregatorConfigSecret(cr *loggingService.LoggingService) error {
	credentials, err := r.resolveAggregatorOutputCredentials(cr)
	if err != nil {
		r.Log.Error(err, "Failed to resolve aggregator output credentials")
		return err
	}

	m, err := aggregatorConfigSecret(cr, r.DynamicParameters, credentials)
	if err != nil {
		r.Log.Error(err, "Failed creating Secret manifest")
		return err
	}

	_, err = r.CreateOrUpdateConfigSecret(cr, m)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Cannot create or update config secret %s", m.Name))
		return err
	}

	if err = r.DeleteLegacyConfigMap(cr.GetNamespace(), util.AggregatorFluentbitComponentName); err != nil {
		r.Log.Error(err, fmt.Sprintf("Cannot delete the legacy config map %s", util.AggregatorFluentbitComponentName))
		return err
	}

	return nil
}

// resolveAggregatorOutputCredentials reads the Secrets referenced by the enabled
// aggregator outputs and returns their values so that they can be inlined into the
// configuration Secret instead of being exposed as environment variables or
// persisted on the CR.
func (r *HAFluentReconciler) resolveAggregatorOutputCredentials(cr *loggingService.LoggingService) (aggregatorOutputCredentials, error) {
	credentials := aggregatorOutputCredentials{}
	if cr.Spec.Fluentbit == nil || cr.Spec.Fluentbit.Aggregator == nil || cr.Spec.Fluentbit.Aggregator.Output == nil {
		return credentials, nil
	}
	output := cr.Spec.Fluentbit.Aggregator.Output
	outputs := make([]util.OutputAuth, 0, 3)
	if output.Loki != nil {
		outputs = append(outputs, util.OutputAuth{Values: &credentials.Loki, Enabled: output.Loki.Enabled, Auth: output.Loki.Auth})
	}
	if output.Http != nil {
		outputs = append(outputs, util.OutputAuth{Values: &credentials.Http, Enabled: output.Http.Enabled, Auth: output.Http.Auth})
	}
	if output.Otel != nil {
		outputs = append(outputs, util.OutputAuth{Values: &credentials.Otel, Enabled: output.Otel.Enabled, Auth: output.Otel.Auth})
	}
	if err := r.ResolveFluentbitOutputAuth(cr.GetNamespace(), outputs...); err != nil {
		return aggregatorOutputCredentials{}, err
	}
	return credentials, nil
}

func (r *HAFluentReconciler) handleAggregatorStatefulSet(cr *loggingService.LoggingService) error {
	ss, err := aggregatorStatefulSet(cr)
	if err != nil {
		r.Log.Error(err, "Failed creating Stateful Set manifest")
		return err
	}

	if err = r.createOrUpdateAggregatorStatefulSet(cr, ss); err != nil {
		return err
	}

	return r.waitForAggregator(cr)
}

func (r *HAFluentReconciler) createOrUpdateAggregatorStatefulSet(cr *loggingService.LoggingService,
	desired *appsv1.StatefulSet) error {
	err := r.CreateResource(cr, desired)
	if err == nil {
		return nil
	}
	if !api_errors.IsAlreadyExists(err) {
		return err
	}

	existing := &appsv1.StatefulSet{ObjectMeta: desired.ObjectMeta}
	if err = r.GetResource(existing); err != nil {
		return err
	}

	if existing.Labels == nil && desired.Labels != nil {
		existing.SetLabels(desired.Labels)
	} else {
		maps.Copy(existing.Labels, desired.Labels)
	}
	existing.Spec.Template.SetLabels(desired.Spec.Template.GetLabels())
	existing.Spec.Template.Spec.Containers = desired.Spec.Template.Spec.Containers
	existing.Spec.Template.Spec.ServiceAccountName = desired.Spec.Template.Spec.ServiceAccountName
	existing.Spec.Template.Spec.NodeSelector = desired.Spec.Template.Spec.NodeSelector
	existing.Spec.Template.Spec.Volumes = desired.Spec.Template.Spec.Volumes
	existing.Spec.Template.Spec.Tolerations = desired.Spec.Template.Spec.Tolerations
	existing.Spec.Template.Spec.Affinity = desired.Spec.Template.Spec.Affinity

	return r.UpdateResource(existing)
}

func (r *HAFluentReconciler) waitForAggregator(cr *loggingService.LoggingService) error {
	// Delay to allow time for the deploy to be updated
	time.Sleep(util.InitialDelay)

	// Wait for Aggregator running
	podManager := util.NewPodManager(r.Client, cr.GetNamespace(), r.Log)
	timeout := util.FluentbitAggregatorPendingTimeout
	if cr.Spec.Fluentbit.Aggregator.StartupTimeout != 0 {
		timeout = time.Duration(cr.Spec.Fluentbit.Aggregator.StartupTimeout) * time.Minute
	}
	started, err := podManager.WaitForStatefulsetUpdated(util.AggregatorFluentbitComponentName, timeout)
	if err != nil {
		return err
	}

	if !started {
		r.StatusUpdater.UpdateStatus(util.HAFluentStatus, util.Failed, false, "Fluent bit aggregator is not started")
		return errors.New("fluent bit aggregator is not started")
	}
	return nil
}

func (r *HAFluentReconciler) handleAggregatorService(cr *loggingService.LoggingService) error {
	m, err := aggregatorService(cr)
	if err != nil {
		r.Log.Error(err, "Failed creating Service manifest")
		return err
	}

	if err = r.CreateResource(cr, m); err != nil {
		if api_errors.IsAlreadyExists(err) {
			e := &corev1.Service{ObjectMeta: m.ObjectMeta}
			if err = r.GetResource(e); err != nil {
				return err
			}

			//Set parameters
			e.SetLabels(m.GetLabels())
			e.Spec.Ports = m.Spec.Ports
			e.Spec.Selector = m.Spec.Selector

			if err = r.UpdateResource(e); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func (r *HAFluentReconciler) deleteDaemonSet(cr *loggingService.LoggingService, name string) error {
	e := &appsv1.DaemonSet{
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

func (r *HAFluentReconciler) deleteStatefulSet(cr *loggingService.LoggingService, name string) error {
	e := &appsv1.StatefulSet{
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

func (r *HAFluentReconciler) deleteConfigMap(cr *loggingService.LoggingService, name string) error {
	e := &corev1.ConfigMap{
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

func (r *HAFluentReconciler) deleteService(cr *loggingService.LoggingService, name string) error {
	e := &corev1.Service{
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

func (r *HAFluentReconciler) updateConfigMap(cr *loggingService.LoggingService, configMap *corev1.ConfigMap) (updated bool, err error) {
	if err = r.CreateResource(cr, configMap); err != nil {
		if api_errors.IsAlreadyExists(err) {
			existedConfigMap := &corev1.ConfigMap{ObjectMeta: configMap.ObjectMeta}
			if err = r.GetResource(existedConfigMap); err != nil {
				return false, err
			}

			if !r.Equal(existedConfigMap, configMap) {
				if err = r.UpdateResource(configMap); err != nil {
					return false, err
				}

				return true, nil
			}

			r.Log.Info("The config map is not changed")
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (r *HAFluentReconciler) deleteSecret(cr *loggingService.LoggingService, name string) error {
	e := &corev1.Secret{
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
