package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

type Configuration struct {
	Port uint16 `json:"Port"`
	userPool string `json:"userPool"`
	clientSecret string `json:"clientSecret"`
	userPoolRegion string `json:"userPoolRegion"`
	AppClientID string `json:"AppClientID"`
}

func main() {

	// Read configuration file
	configFile := flag.String("config", "config.json", "Configuration file")
	flag.Parse()

	config := Configuration{}
	data, err := os.ReadFile(*configFile)
	if err != nil {
		log.Printf("Failed to read configuration file: %v", err)
		os.Exit(1)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Printf("Failed to parse configuration file: %v", err)
		os.Exit(1)
	}

	server := Server{Port: config.Port}
	server.Players = make(map[uint32]*Player)
	if err := server.StartSSHServer(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
