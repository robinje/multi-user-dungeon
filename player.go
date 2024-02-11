package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/ssh"
)

type Player struct {
	PlayerID      uint64
	Index         uint64
	Name          string
	ToPlayer      chan string
	FromPlayer    chan string
	PlayerError   chan error
	Echo          bool
	Prompt        string
	Connection    ssh.Channel
	Server        *Server
	ConsoleWidth  int
	ConsoleHeight int
	CharacterList map[string]uint64
	Character     *Character
	LoginTime     time.Time
}

type PlayerData struct {
	Name          string
	CharacterList map[string]uint64
}

// HandleSSHRequests handles SSH requests from the client
func (p *Player) HandleSSHRequests(requests <-chan *ssh.Request) {
	for req := range requests {
		switch req.Type {
		case "shell":
			req.Reply(true, nil)
		case "pty-req":
			termLen := req.Payload[3]
			w, h := parseDims(req.Payload[termLen+4:])
			p.ConsoleWidth, p.ConsoleHeight = w, h
			req.Reply(true, nil)
		case "window-change":
			w, h := parseDims(req.Payload)
			p.ConsoleWidth, p.ConsoleHeight = w, h
		}
	}
}

func (k *KeyPair) WritePlayer(player *Player) error {
	// Create a PlayerData instance containing only the data to be serialized
	pd := PlayerData{
		Name:          player.Name,
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

	if err == bolt.ErrBucketNotFound {
		return "", nil, fmt.Errorf("player not found")
	}

	if err != nil {
		return "", nil, err
	}

	if playerData == nil {
		return "", nil, fmt.Errorf("player not found")
	}

	// Deserialize the JSON into a PlayerData struct
	var pd PlayerData
	err = json.Unmarshal(playerData, &pd)
	if err != nil {
		return "", nil, err
	}

	return pd.Name, pd.CharacterList, nil
}

func (p *Player) PlayerInput() {
	reader := bufio.NewReader(p.Connection) // Create a bufio.Reader once here
	var inputBuffer bytes.Buffer            // Buffer to accumulate characters into a line

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
		if p.Echo {
			p.ToPlayer <- string(char)
		}

		// Add character to the buffer
		inputBuffer.WriteRune(char)

		// Check if the character is a newline, indicating the end of input
		if char == '\n' || char == '\r' {
			// Send the accumulated line (excluding the newline character) through the FromPlayer channel
			p.FromPlayer <- strings.TrimRight(inputBuffer.String(), "\r\n")

			// Clear the buffer for the next line of input
			inputBuffer.Reset()
		}
	}

	// Close the channel to signify no more input will be processed
	close(p.FromPlayer)
}

func (p *Player) PlayerOutput() {
	for message := range p.ToPlayer {
		// Append carriage return and newline for SSH protocol compatibility
		messageToSend := message + "\r\n"
		if _, err := p.Connection.Write([]byte(messageToSend)); err != nil {
			log.Printf("Failed to send message to player %s: %v", p.Name, err)
			// Consider whether to continue or break based on your error handling policy
			continue
		}
	}

	// Optionally, perform any cleanup here after the channel is closed and loop exits
	log.Printf("Message channel closed for player %s", p.Name)
}
