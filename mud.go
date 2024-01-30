package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	bolt "go.etcd.io/bbolt"
)

type Configuration struct {
	Port           uint16 `json:"Port"`
	UserPoolID     string `json:"UserPoolId"`
	ClientSecret   string `json:"UserPoolClientSecret"`
	UserPoolRegion string `json:"UserPoolRegion"`
	ClientID       string `json:"UserPoolClientId"`
	DataFile       string `json:"DataFile"`
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

	// log.Printf("Configuration loaded: %+v", config)

	server := Server{Port: config.Port, Config: config}
	server.Players = make(map[uint64]*Player)
	server.DataBase.File = config.DataFile
	server.Rooms, err = server.Database.LoadRooms()
	if err != nil {
		log.Fatalf("Failed to load rooms: %v", err)
	}

	// Start the server
	if err := server.StartSSHServer(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
