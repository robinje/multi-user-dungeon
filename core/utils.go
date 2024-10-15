package core

import (
	"math"
	"math/rand"
	"strings"
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
	Logger.Info("Starting auto-save routine...")

	for {
		// Sleep for the configured duration
		time.Sleep(time.Duration(server.AutoSave) * time.Minute)

		Logger.Info("Starting auto-save process...")

		// Save active characters
		if err := SaveActiveCharacters(server); err != nil {
			Logger.Error("Failed to save characters", "error", err)
		} else {
			Logger.Info("Active characters saved successfully")
		}

		// Save active rooms
		if err := SaveActiveRooms(server); err != nil {
			Logger.Error("Failed to save rooms", "error", err)
		} else {
			Logger.Info("Active rooms saved successfully")
		}

		// Save active items
		if err := server.SaveActiveItems(); err != nil {
			Logger.Error("Failed to save items", "error", err)
		} else {
			Logger.Info("Active items saved successfully")
		}

		Logger.Info("Auto-save process completed")
	}
}

// CleanupNilItems removes any nil items from the room's item list.
func (r *Room) CleanupNilItems() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	for id, item := range r.Items {
		if item == nil {
			delete(r.Items, id)
			Logger.Info("Removed nil item from room", "itemID", id, "roomID", r.RoomID)
		}
	}
}

func wrapText(text string, width int) string {
	var result strings.Builder
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if len(line) == 0 {
			result.WriteString("\r\n")
			continue
		}

		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}

		lineLen := 0
		for _, word := range words {
			wordLen := len(word)
			if lineLen+wordLen+1 > width {
				result.WriteString("\r\n")
				lineLen = 0
			} else if lineLen > 0 {
				result.WriteString(" ")
				lineLen++
			}
			result.WriteString(word)
			lineLen += wordLen
		}
		result.WriteString("\r\n")
	}

	return result.String()
}
