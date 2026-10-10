package cmd

import "github.com/spf13/cobra"

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "pst",
	Short: "CLI client for the Postier webserver to send or receive text",
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

func Execute() error {
	return rootCmd.Execute()
}
