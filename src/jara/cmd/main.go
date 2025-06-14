package main

import (
	"os"

	"github.com/bschimke95/jara/pkg/app"
	"github.com/bschimke95/jara/pkg/app/startup"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "jara",
	Short: "JARA - Juju Application Runner and Analyzer",
	Long: `JARA is a TUI application for managing Juju models and applications.
It provides an interactive interface for deploying, scaling, and managing Juju applications.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default behavior when no subcommand is provided
		// This will start the TUI application
		if err := run(); err != nil {
			os.Exit(1)
		}
	},
}

func run() error {
	// Initialize the Bubble Tea program
	initialPage := startup.New()
	model := app.New(initialPage)
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Run the program
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func init() {
	// Initialize Viper for configuration
	viper.SetConfigName("config")
	viper.AddConfigPath("$HOME/.jara")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
