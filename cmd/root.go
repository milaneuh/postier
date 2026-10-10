package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "pst",
	Short: "CLI client for the Postier webserver to send or receive text.",
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Print the latest content saved in the Postier webserver.",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := http.Get("http://localhost:8080/test.txt")

		if err != nil {
			log.Fatal("Unable to fetch the latest clipboard entry.")
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			bodyBytes, err := io.ReadAll(resp.Body)

			if err != nil {
				log.Fatal("Unable to read the clipboard entry.")
			}

			bodyString := string(bodyBytes)
			fmt.Println(bodyString)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.AddCommand(getCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
