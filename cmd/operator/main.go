package main

import (
	"os"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/Netcracker/qubership-logging-operator/internal/manager"
	"github.com/Netcracker/qubership-logging-operator/internal/utils"
)

var setupLog = utils.Logger("cmd")

func main() {
	// The context is canceled on SIGTERM or SIGINT, which stops the manager gracefully.
	if err := manager.RunManager(signals.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "cannot setup manager")
		os.Exit(1)
	}
	setupLog.Info("stopped")
}
