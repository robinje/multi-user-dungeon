package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"time"
)

type Character struct {
	CharacterID uint64
	Room        *Room
	Name        string
	Player      *Player
}

func NewCharacter(CharacterID uint64, Name string, Player *Player, Room *Room) *Character {
	return &Character{
		CharacterID: CharacterID,
		Room:        Room,
		Name:        Name,
		Player:      Player,
	}
}

func (c *Character) SendMessage(message string) {
	c.Player.SendMessage(message)
}

func (c *Character) InputLoop() {
	reader := bufio.NewReader(c.Player.Connection)

	go func() {
		for msg := range c.Player.ToPlayer {
			c.Player.Connection.Write([]byte(msg))
		}
	}()

	c.Player.WritePrompt()

	var inputBuffer bytes.Buffer
	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			if err != io.EOF {
				c.Player.Connection.Write([]byte(fmt.Sprintf("Error: %v\n\r", err)))
			}
			return
		}

		// Echo the character back to the player
		c.Player.Connection.Write([]byte(string(char)))

		// Add character to buffer
		inputBuffer.WriteRune(char)

		// Check if the character is a newline
		if char == '\n' || char == '\r' {
			inputLine := inputBuffer.String()

			// Normalize line ending to \n\r
			inputLine = strings.Replace(inputLine, "\n", "\n\r", -1)

			// Process the command
			verb, tokens, err := validateCommand(strings.TrimSpace(inputLine), valid_commands)
			if err != nil {
				c.Player.Connection.Write([]byte(err.Error() + "\n\r"))
				c.Player.WritePrompt()
				inputBuffer.Reset()
				continue
			}

			if executeCommand(c, verb, tokens) {
				time.Sleep(100 * time.Millisecond)
				inputBuffer.Reset()
				return
			}

			log.Printf("Player %s issued command: %s", c.Player.Name, strings.Join(tokens, " "))
			c.Player.WritePrompt()
			inputBuffer.Reset()
		}
	}
}
