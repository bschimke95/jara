package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bschimke95/jara/internal/config"
)

func infoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Print jara configuration info",
		Long:  "Display the active configuration paths and runtime information for jara.",
		RunE:  printInfo,
	}
}

func printInfo(_ *cobra.Command, _ []string) error {
	const fmat = "%-18s %s\n"

	fmt.Println(logo)
	fmt.Println()
	fmt.Printf(fmat, "Version:", version)
	fmt.Printf(fmat, "Config:", *jaraFlags.ConfigFile)
	fmt.Printf(fmat, "Skins:", config.DefaultSkinDir())
	fmt.Printf(fmat, "Logs:", config.DefaultLogFile())

	return nil
}
