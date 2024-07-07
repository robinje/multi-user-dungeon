package core

import (
	"log"
	"math"
	"math/rand"
	"time"
)

func Challenge(attacker, defender, balance float64) float64 {
	// Calculate the difference to determine the shift
	diff := attacker - defender

	// Simplified sigmoid function evaluation at x=0 with shift
	sigmoidValue := 1 / (1 + math.Exp(balance*diff))

	// Generate a random float64 number
	randomNumber := rand.Float64()

	// Divide the random number by the sigmoid value
	result := randomNumber / sigmoidValue

	return result
}

func AutoSave(server *Server) {
	log.Printf("Starting auto-save routine...")

	for {
		// Sleep for the configured duration
		time.Sleep(time.Duration(server.AutoSave) * time.Minute)

		log.Println("Starting auto-save process...")

		// Save active characters
		if err := SaveActiveCharacters(server); err != nil {
			log.Printf("Failed to save characters: %v", err)
		} else {
			log.Println("Active characters saved successfully")
		}

		// Save active rooms
		if err := SaveActiveRooms(server); err != nil {
			log.Printf("Failed to save rooms: %v", err)
		} else {
			log.Println("Active rooms saved successfully")
		}

		// Save active items
		if err := server.SaveActiveItems(); err != nil {
			log.Printf("Failed to save items: %v", err)
		} else {
			log.Println("Active items saved successfully")
		}

		// Save active player records
		savedPlayers := 0
		server.Mutex.Lock()
		for _, character := range server.Characters {
			if character.Player != nil {
				err := server.Database.WritePlayer(character.Player)
				if err != nil {
					log.Printf("Failed to save player data for %s: %v", character.Player.Name, err)
				} else {
					savedPlayers++
				}
			}
		}
		server.Mutex.Unlock()
		log.Printf("Saved %d active player records", savedPlayers)

		log.Println("Auto-save process completed")
	}
}
