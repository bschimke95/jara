package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/bschimke95/jara/pkg/log"
	"github.com/canonical/k8s/cmd/k8sd"
	"github.com/spf13/cobra"
)

func main() {
	// execution environment
	env := cmdutil.DefaultExecutionEnvironment()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// logging
	ctx = log.NewContext(ctx, log.L())

	// ensure hooks from all commands are executed
	cobra.EnableTraverseRunHooks = true

	err := k8sd.NewRootCmd(env).ExecuteContext(ctx)

	// Although k8s commands typically use Run instead of RunE and handle
	// errors directly within the command execution, this acts as a safeguard in
	// case any are overlooked.
	//
	// Furthermore, the Cobra framework may not invoke the "Run*" entry points
	// at all in case of argument parsing errors, in which case we *need* to
	// handle the errors here.
	if err != nil {
		env.Exit(1)
	}
}
