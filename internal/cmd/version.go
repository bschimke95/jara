package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	var short bool

	command := &cobra.Command{
		Use:   "version",
		Short: "Print version/build info",
		Long:  "Print version/build information for jara.",
		Run: func(_ *cobra.Command, _ []string) {
			printVersion(short)
		},
	}

	command.Flags().BoolVarP(&short, "short", "s", false, "Print version in short format")

	return command
}

func printVersion(short bool) {
	if short {
		fmt.Printf("%s\n", version)
		return
	}

	const fmat = "%-12s %s\n"
	fmt.Println(logo)
	fmt.Println()
	fmt.Printf(fmat, "Version:", version)
	fmt.Printf(fmat, "Commit:", commit)
	fmt.Printf(fmat, "Date:", date)
}

const logo = `      ██╗ █████╗ ██████╗  █████╗
      ██║██╔══██╗██╔══██╗██╔══██╗
      ██║███████║██████╔╝███████║
 ██   ██║██╔══██║██╔══██╗██╔══██║
 ╚█████╔╝██║  ██║██║  ██║██║  ██║
  ╚════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝`
