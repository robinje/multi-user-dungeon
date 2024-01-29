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

	// Log the configuration
	// log.Printf("Configuration loaded: %+v", config)

	server := Server{Port: config.Port, Config: config}
	server.Players = make(map[uint64]*Player)
	server.Rooms, _ = LoadBolt(config.DataFile)
	if err := server.StartSSHServer(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func LoadBolt(fileName string) (map[int64]*Room, error) {

	rooms := make(map[int64]*Room)

	db, err := bolt.Open(fileName, 0600, nil)
	if err != nil {
		fmt.Printf("Error opening BoltDB file: %v\n", err)
		return rooms, fmt.Errorf("error opening BoltDB file: %w", err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		roomsBucket := tx.Bucket([]byte("Rooms"))
		if roomsBucket == nil {
			fmt.Println("Rooms bucket not found")
			return fmt.Errorf("Rooms bucket not found")
		}

		exitsBucket := tx.Bucket([]byte("Exits"))
		if exitsBucket == nil {
			fmt.Println("Exits bucket not found")
			return fmt.Errorf("Exits bucket not found")
		}

		err := roomsBucket.ForEach(func(k, v []byte) error {
			var room Room
			if err := json.Unmarshal(v, &room); err != nil {
				fmt.Printf("Error unmarshalling room data for key %s: %v\n", k, err)
				return fmt.Errorf("error unmarshalling room data: %w", err)
			}
			rooms[int64(room.RoomID)] = &room
			fmt.Println("Loaded Room:", room.RoomID)
			return nil
		})
		if err != nil {
			return err
		}

		return exitsBucket.ForEach(func(k, v []byte) error {
			var exit Exit
			if err := json.Unmarshal(v, &exit); err != nil {
				fmt.Printf("Error unmarshalling exit data for key %s: %v\n", k, err)
				return fmt.Errorf("error unmarshalling exit data: %w", err)
			}

			keyParts := strings.SplitN(string(k), "_", 2)
			if len(keyParts) != 2 {
				fmt.Printf("Invalid exit key format: %s\n", k)
				return fmt.Errorf("invalid exit key format")
			}
			roomID, err := strconv.ParseInt(keyParts[0], 10, 64)
			if err != nil {
				fmt.Printf("Error parsing room ID from key %s: %v\n", k, err)
				return fmt.Errorf("error parsing room ID from key: %w", err)
			}

			if room, exists := rooms[roomID]; exists {
				room.Exits[exit.Direction] = &exit
				// fmt.Printf("Loaded Exit %s for Room %d: %+v\n", exit.Direction, room.RoomID, exit)
			} else {
				fmt.Printf("Room not found for exit key %s\n", k)
				return fmt.Errorf("room not found for exit: %s", string(k))
			}
			return nil
		})
	})

	if err != nil {
		fmt.Printf("Error reading from BoltDB: %v\n", err)
		return rooms, fmt.Errorf("error reading from BoltDB: %w", err)
	}

	// Display(rooms)

	return rooms, nil
}
