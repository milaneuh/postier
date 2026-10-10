package cmd

import (
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the clipboard folder over HTTP",
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("APP_ENV") == "dev" {
			log.Println("We are in dev environment, moving the test clipboard file in our serving folder ...")
			if err := os.Rename("./assets/test.txt", "./clipboard/test.txt"); err != nil {
				log.Println("could not move test file:", err)
			}
		}

		http.Handle("/", http.FileServer(http.Dir("./clipboard")))
		log.Println("Serving on http://localhost:8080")
		return http.ListenAndServe(":8080", nil)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
