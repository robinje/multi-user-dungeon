package main

import (
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

type Player struct {
	PlayerID      uint64
	Index         uint64
	Name          string
	ToPlayer      chan string
	FromPlayer    chan string
	PlayerError   chan error
	Prompt        string
	Connection    ssh.Channel
	Server        *Server
	ConsoleWidth  int
	ConsoleHeight int
	CharacterList map[string]uint64
	Character     *Character
	LoginTime     time.Time
}

// WritePrompt sends the command prompt to the player
func (p *Player) WritePrompt() {
	p.Connection.Write([]byte(p.Prompt))
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

// SendMessage sends a message to the player
func (p *Player) SendMessage(message string) {
	p.ToPlayer <- message
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
