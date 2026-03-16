// Package cmd implements the cobra CLI for jara.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bschimke95/jara/internal/config"
)

var (
	// version, commit, date are set by ldflags at build time.
	version = "dev"
	commit  = "none"
	date    = "unknown"

	jaraFlags *config.Flags

	rootCmd = &cobra.Command{
		Use:   config.AppName,
		Short: "A TUI for Juju cluster management.",
		Long:  "Jara is a terminal user interface (TUI) to observe and interact with Juju clusters, inspired by k9s.",
		RunE:  run,
		// Silence default usage/error printing so we control it.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd(), infoCmd())
	initJaraFlags()
}

// Execute runs the root cobra command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initJaraFlags() {
	jaraFlags = config.NewFlags()

	// Config file flag.
	cfgFile := config.DefaultConfigFile()
	jaraFlags.ConfigFile = &cfgFile
	rootCmd.PersistentFlags().StringVar(
		jaraFlags.ConfigFile,
		"config",
		cfgFile,
		"Path to the jara configuration file",
	)

	// Refresh rate.
	var refreshRate float64
	jaraFlags.RefreshRate = &refreshRate
	rootCmd.Flags().Float64VarP(
		jaraFlags.RefreshRate,
		"refresh", "r",
		0, // 0 means "use config file value or default"
		fmt.Sprintf("Status poll interval in seconds (default %.0f)", config.DefaultRefreshRate),
	)

	// Log level.
	var logLevel string
	jaraFlags.LogLevel = &logLevel
	rootCmd.Flags().StringVarP(
		jaraFlags.LogLevel,
		"logLevel", "l",
		"",
		"Log level: error, warn, info, debug (default from config or \"info\")",
	)

	// Log file.
	var logFile string
	jaraFlags.LogFile = &logFile
	rootCmd.Flags().StringVar(
		jaraFlags.LogFile,
		"logFile",
		"",
		"Path to the log file (default from config or cache dir)",
	)

	// Headless.
	var headless bool
	jaraFlags.Headless = &headless
	rootCmd.Flags().BoolVar(
		jaraFlags.Headless,
		"headless",
		false,
		"Hide the header panel",
	)

	// Logoless.
	var logoless bool
	jaraFlags.Logoless = &logoless
	rootCmd.Flags().BoolVar(
		jaraFlags.Logoless,
		"logoless",
		false,
		"Hide the logo in the header",
	)

	// Read-only.
	var readOnly bool
	jaraFlags.ReadOnly = &readOnly
	rootCmd.Flags().BoolVar(
		jaraFlags.ReadOnly,
		"readonly",
		false,
		"Disable write operations",
	)

	// Command.
	var command string
	jaraFlags.Command = &command
	rootCmd.Flags().StringVarP(
		jaraFlags.Command,
		"command", "c",
		"",
		"Override the default view/command on launch",
	)
}
