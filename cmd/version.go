package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version       = "dev"
	defaultServer = "http://localhost:8080"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version variables",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("version: " + version)
		fmt.Println("defaultServer: " + defaultServer)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
