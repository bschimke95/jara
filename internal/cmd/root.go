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

	// Config file flag — always set since it has a meaningful default.
	cfgFile := config.DefaultConfigFile()
	jaraFlags.ConfigFile = &cfgFile
	rootCmd.PersistentFlags().StringVar(
		jaraFlags.ConfigFile,
		"config",
		cfgFile,
		"Path to the jara configuration file",
	)

	// Refresh rate.
	rootCmd.Flags().Float64P(
		"refresh", "r",
		0, // 0 means "use config file value or default"
		fmt.Sprintf("Status poll interval in seconds (default %.0f)", config.DefaultRefreshRate),
	)

	// Log level.
	rootCmd.Flags().StringP(
		"logLevel", "l",
		"",
		"Log level: error, warn, info, debug (default from config or \"info\")",
	)

	// Log file.
	rootCmd.Flags().String(
		"logFile",
		"",
		"Path to the log file (default from config or cache dir)",
	)

	// Headless.
	rootCmd.Flags().Bool(
		"headless",
		false,
		"Hide the header panel",
	)

	// Logoless.
	rootCmd.Flags().Bool(
		"logoless",
		false,
		"Hide the logo in the header",
	)

	// Read-only.
	rootCmd.Flags().Bool(
		"readonly",
		false,
		"Disable write operations",
	)

	// Command.
	rootCmd.Flags().StringP(
		"command", "c",
		"",
		"Override the default view/command on launch",
	)

	// Demo mode (hidden) — use MockClient instead of a real Juju connection.
	rootCmd.Flags().Bool(
		"demo",
		false,
		"Use synthetic mock data instead of a live Juju connection",
	)
	_ = rootCmd.Flags().MarkHidden("demo")
}

// resolveFlagsFrom populates jaraFlags pointers only for flags that were
// explicitly set on the command line. This ensures config-file values
// are not overridden by zero-valued CLI defaults.
func resolveFlagsFrom(cmd *cobra.Command) {
	flags := cmd.Flags()

	if flags.Changed("refresh") {
		v, _ := flags.GetFloat64("refresh")
		jaraFlags.RefreshRate = &v
	}
	if flags.Changed("logLevel") {
		v, _ := flags.GetString("logLevel")
		jaraFlags.LogLevel = &v
	}
	if flags.Changed("logFile") {
		v, _ := flags.GetString("logFile")
		jaraFlags.LogFile = &v
	}
	if flags.Changed("headless") {
		v, _ := flags.GetBool("headless")
		jaraFlags.Headless = &v
	}
	if flags.Changed("logoless") {
		v, _ := flags.GetBool("logoless")
		jaraFlags.Logoless = &v
	}
	if flags.Changed("readonly") {
		v, _ := flags.GetBool("readonly")
		jaraFlags.ReadOnly = &v
	}
	if flags.Changed("command") {
		v, _ := flags.GetString("command")
		jaraFlags.Command = &v
	}
	if flags.Changed("demo") {
		v, _ := flags.GetBool("demo")
		jaraFlags.Demo = &v
	}
}
