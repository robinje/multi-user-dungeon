package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	bolt "go.etcd.io/bbolt"
)

func (k *KeyPair) WritePlayer(player *Player) error {
	// Create a PlayerData instance containing only the data to be serialized
	pd := PlayerData{
		Name:          player.PlayerID,
		CharacterList: player.CharacterList,
	}

	// Serialize the PlayerData struct to JSON
	playerData, err := json.Marshal(pd)
	if err != nil {
		return err
	}

	// Use the player's Name as the key to store the serialized data
	return k.Put("Players", []byte(player.Name), playerData)
}

func (k *KeyPair) ReadPlayer(playerName string) (string, map[string]uint64, error) {
	playerData, err := k.Get("Players", []byte(playerName))
	if err != nil {
		if err == bolt.ErrBucketNotFound {
			log.Println("Player bucket not found")
			return "", nil, fmt.Errorf("player not found")
		}
		log.Printf("Error reading player data: %v", err)
		return "", nil, fmt.Errorf("database read failed: %w", err)
	}

	if playerData == nil {
		log.Printf("Player %s not found", playerName)
		return "", nil, fmt.Errorf("player not found")
	}

	var pd PlayerData
	if err := json.Unmarshal(playerData, &pd); err != nil {
		log.Printf("Error unmarshalling player data: %v", err)
		return "", nil, fmt.Errorf("unmarshal player data: %w", err)
	}

	return pd.Name, pd.CharacterList, nil
}

func PlayerInput(p *Player) {

	log.Printf("Player %s input goroutine started", p.Name)

	var inputBuffer bytes.Buffer

	reader := bufio.NewReader(p.Connection)

	for {
		char, _, err := reader.ReadRune() // Read one rune (character) at a time from the buffered reader
		if err != nil {
			if err == io.EOF {
				// Handle EOF to indicate client disconnect gracefully
				log.Printf("Player %s disconnected: %v", p.Name, err)
				p.PlayerError <- err
				break // Exit the loop on EOF
			} else {
				// Log and handle other errors without breaking the loop
				log.Printf("Error reading from player %s: %v", p.Name, err)
				p.PlayerError <- err
				continue
			}
		}

		// Echo the character back to the player if Echo is true
		// Ensure we do not echo back newline characters, maintaining input cleanliness
		if p.Echo && char != '\n' && char != '\r' {
			if _, err := p.Connection.Write([]byte(string(char))); err != nil {
				log.Printf("Failed to echo character to player %s: %v", p.Name, err)
			}
		}

		// Check if the character is a newline, indicating the end of input
		if char == '\n' || char == '\r' {
			// Trim the newline character and send the input through the FromPlayer channel
			// This assumes that the inputBuffer contains the input line up to the newline character
			if inputBuffer.Len() > 0 { // Ensure we have something to send
				p.FromPlayer <- inputBuffer.String()
				inputBuffer.Reset() // Clear the buffer for the next line of input
			}
			continue
		}

		// Add character to the buffer for accumulating the line
		inputBuffer.WriteRune(char)
	}

	// Close the channel to signify no more input will be processed
	close(p.FromPlayer)
}

func PlayerOutput(p *Player) {

	log.Printf("Player %s output goroutine started", p.Name)

	for message := range p.ToPlayer {
		// Append carriage return and newline for SSH protocol compatibility
		messageToSend := message
		if _, err := p.Connection.Write([]byte(messageToSend)); err != nil {
			log.Printf("Failed to send message to player %s: %v", p.Name, err)
			// Consider whether to continue or break based on your error handling policy
			continue
		}
	}

	// Optionally, perform any cleanup here after the channel is closed and loop exits
	log.Printf("Message channel closed for player %s", p.Name)
}

func InputLoop(c *Character) {

	log.Printf("Starting input loop for character %s", c.Name)

	// Initially execute the look command with no additional tokens
	ExecuteLookCommand(c, []string{}) // Adjusted to include the tokens parameter

	// Send initial prompt to player
	c.Player.ToPlayer <- c.Player.Prompt

	for {
		// Wait for input from the player. This blocks until input is received.
		inputLine, more := <-c.Player.FromPlayer
		if !more {
			// If the channel is closed, stop the input loop.
			log.Printf("Input channel closed for player %s.", c.Player.Name)
			return
		}

		// Normalize line ending to \n\r for consistency
		inputLine = strings.Replace(inputLine, "\n", "\n\r", -1)

		// Process the command
		verb, tokens, err := ValidateCommand(strings.TrimSpace(inputLine))
		if err != nil {
			c.Player.ToPlayer <- err.Error() + "\n\r"
			c.Player.ToPlayer <- c.Player.Prompt
			continue
		}

		// Execute the command
		if ExecuteCommand(c, verb, tokens) {
			// If command execution indicates to exit (or similar action), break the loop
			// Note: Adjust logic as per your executeCommand's design to handle such conditions
			break
		}

		// Log the command execution
		log.Printf("Player %s issued command: %s", c.Player.Name, strings.Join(tokens, " "))

		// Prompt for the next command
		c.Player.ToPlayer <- c.Player.Prompt
	}

	// Close the player's input channel
	close(c.Player.FromPlayer)

	// Remove the character from the room

	c.Room.Mutex.Lock()
	delete(c.Room.Characters, c.Index)
	c.Room.Mutex.Unlock()

	// Remove the character from the server's active characters
	c.Server.Mutex.Lock()
	delete(c.Server.Characters, c.Name)
	c.Server.Mutex.Unlock()

	// Save the character to the database
	err := c.Server.Database.WriteCharacter(c)
	if err != nil {
		log.Printf("Error saving character %s: %v", c.Name, err)
	}
}
