package manager

import (
	"context"
	"flag"
	"fmt"
	"os"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/controllers"
	"github.com/Netcracker/qubership-logging-operator/internal/utils"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
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
	fmt.Fprintf(os.Stdout, "Go Version: %s, Go OS/Arch: %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func RunManager(ctx context.Context) error {
	if err := managerFlags.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("cannot parse provided flags: %w", err)
	}

	printVersion()

	logf.SetLogger(logger)

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	klog.SetLogger(logger)
	ctrl.SetLogger(logger)

	namespace, found := os.LookupEnv("WATCH_NAMESPACE")
	if !found {
		namespace = "logging"
	}

	metricsServerOptions := metricsserver.Options{
		BindAddress: *metricsAddr,
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Logger:                  logger,
		Scheme:                  scheme,
		Metrics:                 metricsServerOptions,
		HealthProbeBindAddress:  *probeAddr,
		PprofBindAddress:        *pprofAddr,
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
		logger.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("ready", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if err = (&controllers.LoggingServiceReconciler{
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		Log:                      utils.Logger("controller-loggingservice"),
		Config:                   mgr.GetConfig(),
		TimeoutOnFailedReconcile: controllers.InitialTimeoutOnFailedReconcile,
		DynamicParameters:        utils.DynamicParameters{ContainerRuntimeType: ""},
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller", "controller", "LoggingService")
		os.Exit(1)
	}

	logger.Info("starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error(err, "problem running manager")
		os.Exit(1)
	}
	return nil
}
