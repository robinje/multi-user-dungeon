package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Character struct {
	Index  uint64
	Room   *Room
	Name   string
	Player *Player
	Mutex  sync.Mutex
	Server *Server
}

// CharacterData for unmarshalling character.
type CharacterData struct {
	Index  uint64 `json:"index"`
	Name   string `json:"name"`
	RoomID int64  `json:"roomID"`
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

	room, ok := s.Rooms[1] //TODO: This should be a function to get the starting room
	if !ok {
		room, ok = s.Rooms[0]
		if !ok {
			return nil, fmt.Errorf("no room found")
		}
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

	// Generate a new unique index for the character
	characterIndex, err := s.Database.NextIndex("Characters")
	if err != nil {
		log.Printf("Error generating character index: %v", err)
		return nil
	}

	character := &Character{
		Index:  characterIndex,
		Room:   Room,
		Name:   Name,
		Player: Player,
		Server: s,
	}

	err = s.WriteCharacter(character)
	if err != nil {
		log.Printf("Error writing character to database: %v", err)
		return nil
	}

	log.Printf("Created character %s with Index %d", character.Name, character.Index)

	Player.CharacterList[Name] = characterIndex

	err = s.Database.WritePlayer(Player)
	if err != nil {
		log.Printf("Error writing player to database: %v", err)
		return nil

	}

	return character
}

func (s *Server) WriteCharacter(character *Character) error {
	characterData, err := json.Marshal(character)
	if err != nil {
		log.Printf("Error marshalling character data: %v", err)
		return err
	}

	err = s.Database.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("Characters"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		indexKey := fmt.Sprintf("%d", character.Index)
		err = bucket.Put([]byte(indexKey), characterData)
		if err != nil {
			return fmt.Errorf("failed to put character data: %v", err)
		}
		return nil
	})

	if err != nil {
		log.Printf("Failed to add character to database: %v", err)
		return err
	}

	log.Printf("Successfully added character %s with Index %d to database", character.Name, character.Index)
	return nil
}

func (s *Server) LoadCharacter(player *Player, characterName string) (*Character, error) {
	var characterData []byte
	err := s.Database.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("Characters"))
		if bucket == nil {
			return fmt.Errorf("characters bucket not found")
		}
		characterData = bucket.Get([]byte(characterName))
		return nil
	})

	if err != nil {
		return nil, err
	}

	if characterData == nil {
		return nil, fmt.Errorf("character not found")
	}

	var cd CharacterData
	if err := json.Unmarshal(characterData, &cd); err != nil {
		return nil, fmt.Errorf("error unmarshalling character data: %w", err)
	}

	// Retrieve the Room object based on RoomID
	room, exists := s.Rooms[cd.RoomID]
	if !exists {
		return nil, fmt.Errorf("room with ID %d not found", cd.RoomID)
	}

	character := &Character{
		Index:  cd.Index,
		Room:   room,
		Name:   cd.Name,
		Player: player,
		Server: s,
	}

	log.Printf("Loaded and created character %s with Index %d in Room %d", character.Name, character.Index, room.RoomID)

	return character, nil
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

	// Initially execute the look command with no additional tokens
	executeLookCommand(c, []string{}) // Adjusted to include the tokens parameter

	time.Sleep(100 * time.Millisecond)

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
			verb, tokens, err := validateCommand(strings.TrimSpace(inputLine), validCommands) // Corrected to validCommands
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
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if c.Room == nil {
		c.SendMessage("You are not in any room to move from.\n\r")
		return
	}

	log.Printf("Player %s is moving %s", c.Name, direction)

	selectedExit, exists := c.Room.Exits[direction]
	if !exists {
		c.SendMessage("You cannot go that way.\n\r")
		return
	}

	newRoom, exists := c.Server.Rooms[selectedExit.TargetRoom]
	if !exists {
		c.SendMessage("The path leads nowhere.\n\r")
		return
	}

	// Safely remove the character from the old room
	oldRoom := c.Room
	oldRoom.Mutex.Lock()
	delete(oldRoom.Characters, c.Index)
	oldRoom.Mutex.Unlock()
	oldRoom.SendRoomMessage(fmt.Sprintf("\n\r%s has left going %s.\n\r", c.Name, direction))

	// Update character's room
	c.Room = newRoom

	newRoom.SendRoomMessage(fmt.Sprintf("\n\r%s has arrived.\n\r", c.Name))

	// Ensure the Characters map in the new room is initialized
	newRoom.Mutex.Lock()
	if newRoom.Characters == nil {
		newRoom.Characters = make(map[uint64]*Character)
	}
	newRoom.Characters[c.Index] = c
	newRoom.Mutex.Unlock()

	executeLookCommand(c, []string{})
}

func (s *Server) SelectCharacter(player *Player) (*Character, error) {
	var options []string // To store character names for easy reference by index

	for {
		player.SendMessage("Select a character:\n")

		if len(player.CharacterList) > 0 {
			i := 1
			for name := range player.CharacterList {
				player.SendMessage(fmt.Sprintf("%d: %s\n", i, name))
				options = append(options, name) // Append character name to options
				i++
			}
		}
		player.SendMessage("0: Create a new character\n")

		reader := bufio.NewReader(player.Connection)
		input, err := reader.ReadString('\n')
		if err != nil {
			player.Connection.Write([]byte(fmt.Sprintf("Error reading input: %v\n\r", err)))
			return nil, err
		}
		input = strings.TrimSpace(input)

		// Convert input to integer
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 0 || choice > len(options) {
			player.SendMessage("Invalid choice. Please select a valid option.\n")
			continue // Prompt again
		}

		if choice == 0 {
			// Create a new character
			return s.CreateCharacter(player)
		} else {
			// Load an existing character
			characterName := options[choice-1] // Adjust for 0-index; choice-1 maps to the correct character name
			return s.LoadCharacter(player, characterName)
		}
	}
}
