package main

import (
	"log"
)


func main() {
	// Create and start the server
	server := Server{Port: 9050}
	if err := server.StartTelnetServer(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
