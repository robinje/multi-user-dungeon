package main

import (
	"log"
)

func main() {
	server := Server{Port: 9050}
	server.Players = make(map[uint32]*Player)
	if err := server.StartSSHServer(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
