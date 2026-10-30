package manager

import (
	"context"
	"flag"
	"fmt"
	"os"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/controllers"
	"github.com/Netcracker/qubership-logging-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"runtime"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	//+kubebuilder:scaffold:imports
)

var (
	managerFlags       = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	logger             = utils.Logger("cmd")
	metricsPort  int32 = 8080

	scheme = apiruntime.NewScheme()

	metricsAddr  = managerFlags.String("metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	probeAddr    = managerFlags.String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	pprofEnabled = managerFlags.Bool("pprof-enable", true, "Enable pprof.")
	pprofAddr    = managerFlags.String("pprof-address", ":9180", "The pprof address.")

	leaderElection = managerFlags.Bool("leader-elect", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	leaderElectNamespace = managerFlags.String("leader-elect-namespace", "",
		"Defines optional namespace name in which the leader election resource will be created. By default, uses in-cluster namespace name.")
	leaderElectionID = managerFlags.String("leader-election-id", "asd23ha2.logging.netcracker.com",
		"The ID to use for leader election.")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(loggingService.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func printVersion() {
	logger.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	logger.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

// RunManager parses the operator flags, starts the controller manager, and blocks
// until ctx is canceled or the manager fails.
func RunManager(ctx context.Context) error {
	if err := managerFlags.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("cannot parse provided flags: %w", err)
	}

	printVersion()

	logf.SetLogger(logger)
	klog.SetLogger(logger)
	ctrl.SetLogger(logger)

	namespace, found := os.LookupEnv("WATCH_NAMESPACE")
	if !found {
		namespace = "logging"
	}

	pprofBindAddress := ""
	if *pprofEnabled {
		pprofBindAddress = *pprofAddr
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Logger:                  logger,
		Scheme:                  scheme,
		Metrics:                 metricsserver.Options{BindAddress: *metricsAddr},
		HealthProbeBindAddress:  *probeAddr,
		PprofBindAddress:        pprofBindAddress,
		ReadinessEndpointName:   "/ready",
		LivenessEndpointName:    "/health",
		LeaderElection:          *leaderElection,
		LeaderElectionNamespace: *leaderElectNamespace,
		LeaderElectionID:        *leaderElectionID,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("unable to create manager: %w", err)
	}

	if err = mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err = mgr.AddReadyzCheck("ready", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	if err = (&controllers.LoggingServiceReconciler{
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		Log:                      utils.Logger("controller-loggingservice"),
		Config:                   mgr.GetConfig(),
		TimeoutOnFailedReconcile: controllers.InitialTimeoutOnFailedReconcile,
		DynamicParameters:        utils.DynamicParameters{ContainerRuntimeType: ""},
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create the LoggingService controller: %w", err)
	}

	createMetricsPodMonitor(mgr.GetConfig(), namespace)

	logger.Info("starting manager")
	if err = mgr.Start(ctx); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}
	return nil
}

// createMetricsPodMonitor creates the PodMonitor that lets Prometheus scrape the
// operator metrics. Set SKIP_METRICS_SERVICE to "true" to skip it. A failure is logged
// and does not stop the operator.
func createMetricsPodMonitor(config *rest.Config, namespace string) {
	if os.Getenv("SKIP_METRICS_SERVICE") == "true" {
		logger.Info("Skip a creating Service and PodMonitor which scrape metrics from this operator")
		return
	}
	serviceName := "logging-operator-metrics"
	labels := utils.MergeLabels(
		utils.ResourceLabels(serviceName, "logging-operator"),
		map[string]string{"app.kubernetes.io/instance": utils.GetInstanceLabel(serviceName, namespace)},
	)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port:       metricsPort,
				Name:       "http-metrics",
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt32(metricsPort),
			}},
			Selector: labels,
		},
	}
	err := utils.CreatePodMonitors(config, namespace, []*corev1.Service{service},
		os.Getenv("SCRAPE_INTERVAL"), os.Getenv("SCRAPE_TIMEOUT"))
	if err != nil {
		logger.Info("Could not create PodMonitor object", "error", err.Error())
	}
}
