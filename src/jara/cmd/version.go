package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of JARA",
	Long:  `All software has versions. This is JARA's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("JARA Version 0.1.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
