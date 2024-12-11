package main

import (
	"github.com/canonical/k8s/pkg/k8sd/app"
	"github.com/canonical/k8s/pkg/log"
	"github.com/spf13/cobra"
)

var rootCmdOpts struct {
	logLevel int
}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jara",
		Short: "Juju management at your fingertips",
		Run: func(cmd *cobra.Command, args []string) {
			// configure logging
			log.Configure(log.Options{
				LogLevel:     rootCmdOpts.logLevel,
				AddDirHeader: true,
			})

			app, err := app.New(app.Config{
				Debug:                               rootCmdOpts.logDebug,
				Verbose:                             rootCmdOpts.logVerbose,
				StateDir:                            rootCmdOpts.stateDir,
				Snap:                                env.Snap,
				PprofAddress:                        rootCmdOpts.pprofAddress,
				DisableNodeConfigController:         rootCmdOpts.disableNodeConfigController,
				DisableControlPlaneConfigController: rootCmdOpts.disableControlPlaneConfigController,
				DisableUpdateNodeConfigController:   rootCmdOpts.disableUpdateNodeConfigController,
				DisableFeatureController:            rootCmdOpts.disableFeatureController,
				DisableCSRSigningController:         rootCmdOpts.disableCSRSigningController,
			})
			if err != nil {
				cmd.PrintErrf("Error: Failed to initialize k8sd: %v", err)
				env.Exit(1)
				return
			}

			if err := app.Run(cmd.Context(), nil); err != nil {
				cmd.PrintErrf("Error: Failed to run k8sd: %v", err)
				env.Exit(1)
				return
			}
		},
	}

	cmd.SetIn(env.Stdin)
	cmd.SetOut(env.Stdout)
	cmd.SetErr(env.Stderr)

	cmd.PersistentFlags().IntVarP(&rootCmdOpts.logLevel, "log-level", "l", 0, "k8sd log level")

	return cmd
}
