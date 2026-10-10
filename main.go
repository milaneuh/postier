package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	log.Println("Hello World!")

	if os.Getenv("APP_ENV") == "dev" {
		log.Println("We are in dev environment, moving the test clipboard file in our serving folder ...")
		if err := os.Rename("./assets/test.txt", "./clipboard/test.txt"); err != nil {
			log.Println("could not move test file:", err)
		}
	}

	http.Handle("/", http.FileServer(http.Dir("./clipboard")))
	log.Println("Serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
