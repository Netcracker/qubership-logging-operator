package main

import (
	"context"
	"os"
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/Netcracker/qubership-logging-operator/internal/manager"
	"github.com/Netcracker/qubership-logging-operator/internal/utils"
)

var (
	setupLog = utils.Logger("cmd")
)

func main() {
	ctx, cancel := context.WithCancelCause(context.Background())
	stop := signals.SetupSignalHandler()
	go func() {
		<-stop.Done()
		cancel(errors.New("graceful shutdown, exiting"))
	}()

	err := manager.RunManager(ctx)
	if err != nil {
		setupLog.Error(err, "cannot setup manager")
		os.Exit(1)
	}
	setupLog.Info("stopped")
}
