package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"
)

type Configuration struct {
	Port           uint16  `json:"Port"`
	UserPoolID     string  `json:"UserPoolId"`
	ClientSecret   string  `json:"UserPoolClientSecret"`
	UserPoolRegion string  `json:"UserPoolRegion"`
	ClientID       string  `json:"UserPoolClientId"`
	DataFile       string  `json:"DataFile"`
	Balance        float64 `json:"Balance"`
	AutoSave       uint16  `json:"AutoSave"`
}

func main() {
	// Read configuration file
	configFile := flag.String("config", "config.json", "Configuration file")
	flag.Parse()

	config, err := loadConfiguration(*configFile)
	if err != nil {
		log.Printf("Error loading configuration: %v", err)
		return
	}

	server, err := NewServer(config)
	if err != nil {
		log.Printf("Failed to create server: %v", err)
		return
	}

	// Start the auto-save routine
	go server.AutoSaveCharacters()

	// Start the server
	if err := server.StartSSHServer(); err != nil {
		log.Printf("Failed to start server: %v", err)
		return
	}
}

func loadConfiguration(configFile string) (Configuration, error) {
	var config Configuration

	data, err := os.ReadFile(configFile)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func (s *Server) AutoSaveCharacters() {
	for {
		// Sleep for the configured duration
		time.Sleep(time.Duration(s.AutoSave) * time.Minute)

		// Save the characters to the database
		if err := s.SaveActiveCharacters(); err != nil {
			log.Printf("Failed to save characters: %v", err)
		}
	}
}
