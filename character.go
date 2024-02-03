package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"
)

type Character struct {
	Index  uint64
	Room   *Room
	Name   string
	Player *Player
	Mutex  sync.Mutex
	Server *Server
}

func (s *Server) CreateCharacter(player *Player) (*Character, error) {
	reader := bufio.NewReader(player.Connection)

	go func() {
		for msg := range player.ToPlayer {
			player.Connection.Write([]byte(msg))
		}
	}()

	// Prompt the player for the character name
	player.SendMessage("Enter your character name: ")

	var inputBuffer bytes.Buffer
	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			if err != io.EOF {
				player.Connection.Write([]byte(fmt.Sprintf("Error: %v\n\r", err)))
			}
			return nil, err
		}

		// Echo the character back to the player
		player.Connection.Write([]byte(string(char)))

		// Check if the character is a newline, indicating the end of input
		if char == '\n' || char == '\r' {
			break // Exit the loop once the newline character is encountered
		}

		// Add character to buffer
		inputBuffer.WriteRune(char)
	}

	// Trim space to remove the newline character at the end
	charName := strings.TrimSpace(inputBuffer.String())

	player.SendMessage(fmt.Sprintf("\n\r"))

	// Retrieve room 1, or handle the case where it does not exist

	room, ok := s.Rooms[1]
	if !ok {
		return nil, fmt.Errorf("Starting room does not exist")
	}

	log.Printf("Starting room: %v", room)

	// Create and initialize the new character
	character := s.NewCharacter(charName, player, room)

	// Ensure the Characters map is initialized <- Do not like.
	if room.Characters == nil {
		room.Characters = make(map[uint64]*Character)
	}

	// Add the character to the room's Characters map
	room.Characters[character.Index] = character

	return character, nil
}

func (s *Server) NewCharacter(Name string, Player *Player, Room *Room) *Character {
	character := &Character{
		Index:  s.CharacterIndex.GetID(),
		Room:   Room,
		Name:   Name,
		Player: Player,
		Server: s,
	}

	log.Printf("Created character %s with Index %d", character.Name, character.Index)

	return character
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

func (c *Character) Move(direction string) {
	if c.Room == nil {
		c.SendMessage("You must be in a room to move.\n\r")
		return
	}

	oldRoom := c.Room

	selectedExit := c.Room.Exits[direction]
	if selectedExit == nil {
		c.SendMessage("There is no exit in that direction.\n\r")
		return
	}

	newRoom := c.Server.Rooms[selectedExit.TargetRoom]
	if newRoom == nil {
		c.SendMessage("There is no exit in that direction.\n\r")
		return
	}

	// Remove the character from the old room
	oldRoom.Mutex.Lock()
	delete(oldRoom.Characters, c.Index)
	oldRoom.Mutex.Unlock()

	// Send a message to the old room
	oldRoom.SendRoomMessage(c.Name + " has left the room.\n\r")

	c.SendMessage("You move " + direction + ".\n\r")

	// Send a message to the new room
	newRoom.SendRoomMessage(c.Name + " has entered the room.\n\r")

	// Add the character to the new room
	newRoom.Mutex.Lock()
	newRoom.Characters[c.Index] = c
	newRoom.Mutex.Unlock()

	c.Room = newRoom
	executeLookCommand(c)

}
