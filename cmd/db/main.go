package main

import (
	"log"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("database UI available at http://localhost:%s", port)
	log.Fatal(httpServer(port))
}
