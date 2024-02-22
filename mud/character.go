package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"

	bolt "go.etcd.io/bbolt"
)

type Character struct {
	Index      uint64
	Room       *Room
	Name       string
	Player     *Player
	Attributes map[string]float64
	Abilities  map[string]float64
	Mutex      sync.Mutex
	Server     *Server
}

// CharacterData for unmarshalling character.
type CharacterData struct {
	Index      uint64             `json:"index"`
	Name       string             `json:"name"`
	RoomID     int64              `json:"roomID"`
	Attributes map[string]float64 `json:"attributes"`
	Abilities  map[string]float64 `json:"abilities"`
}

type Archetype struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Attributes  map[string]float64 `json:"Attributes"`
	Abilities   map[string]float64 `json:"Abilities"`
}

type ArchetypesData struct {
	Archetypes map[string]Archetype `json:"archetypes"`
}

func (s *Server) SelectCharacter(player *Player) (*Character, error) {
	var options []string // To store character names for easy reference by index

	sendCharacterOptions := func() {
		player.ToPlayer <- "Select a character:\n\r"
		player.ToPlayer <- "0: Create a new character\n\r"

		if len(player.CharacterList) > 0 {
			i := 1
			for name := range player.CharacterList {
				player.ToPlayer <- fmt.Sprintf("%d: %s\n\r", i, name)
				options = append(options, name) // Append character name to options
				i++
			}
		}
		player.ToPlayer <- "Enter the number of your choice: "
	}

	for {
		// Send options to the player
		sendCharacterOptions()

		// Wait for input from the player
		input, ok := <-player.FromPlayer
		if !ok {
			// Handle the case where the channel is closed unexpectedly
			return nil, fmt.Errorf("failed to receive input")
		}

		// Convert input to integer
		choice, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil || choice < 0 || choice > len(options) {
			player.ToPlayer <- "Invalid choice. Please select a valid option.\n\r"
			continue
		}

		if choice == 0 {
			// Create a new character
			return s.CreateCharacter(player)
		} else if choice <= len(options) {
			// Load an existing character, adjusting index for 0-based slice indexing
			characterName := options[choice-1]
			return s.LoadCharacter(player, player.CharacterList[characterName])
		}
	}
}

func (s *Server) CreateCharacter(player *Player) (*Character, error) {
	// Prompt the player for the character name
	player.ToPlayer <- "\n\rEnter your character name: "

	// Wait for input from the player
	charName, ok := <-player.FromPlayer
	if !ok {
		// Handle the case where the channel is closed unexpectedly
		return nil, fmt.Errorf("failed to receive character name input")
	}

	charName = strings.TrimSpace(charName)

	if len(charName) == 0 {
		return nil, fmt.Errorf("character name cannot be empty")
	}

	// Limit character names to 15 characters
	if len(charName) > 15 {
		return nil, fmt.Errorf("character name must be 15 characters or fewer")
	}

	// Check if the character name already exists
	if s.CharacterExists[strings.ToLower(charName)] {
		return nil, fmt.Errorf("character already exists")
	}

	// Check if any archetypes are loaded
	var selectedArchetype string
	if s.Archetypes != nil && len(s.Archetypes.Archetypes) > 0 {
		for {
			// Prepare and send the selection message to the player
			selectionMsg := "\n\rSelect a character archetype.\n\r"
			archetypeOptions := make([]string, 0, len(s.Archetypes.Archetypes))
			for name, archetype := range s.Archetypes.Archetypes {
				// Adding each archetype name and description to the list
				archetypeOptions = append(archetypeOptions, name+" - "+archetype.Description)
			}
			// Optional: Sort the options if needed
			sort.Strings(archetypeOptions)

			for i, option := range archetypeOptions {
				selectionMsg += fmt.Sprintf("%d: %s\n\r", i+1, option)
			}

			selectionMsg += "Enter the number of your choice: "
			// Send the selection message to the player
			player.ToPlayer <- selectionMsg

			// Wait for input from the player
			selection, ok := <-player.FromPlayer
			if !ok {
				return nil, fmt.Errorf("failed to receive archetype selection")
			}

			// Convert selection to an integer and validate
			selectionNum, err := strconv.Atoi(strings.TrimSpace(selection))
			if err == nil && selectionNum >= 1 && selectionNum <= len(archetypeOptions) {
				selectedOption := archetypeOptions[selectionNum-1]
				selectedArchetype = strings.Split(selectedOption, " - ")[0]
				break // Valid selection; break out of the loop
			} else {
				player.ToPlayer <- "Invalid selection. Please select a valid archetype number."
			}
		}
	}

	// Log the character creation attempt
	log.Printf("Creating character with name: %s", charName)

	// Retrieve the starting room for the new character
	room, ok := s.Rooms[1] // Assuming room 1 is the starting room
	if !ok {
		room, ok = s.Rooms[0] // Fallback to room 0 if room 1 doesn't exist
		if !ok {
			return nil, fmt.Errorf("no starting room found")
		}
	}

	// Create and initialize the new character
	character := s.NewCharacter(charName, player, room, selectedArchetype)

	// Ensure the Characters map in the starting room is initialized
	if room.Characters == nil {
		room.Characters = make(map[uint64]*Character)
	}

	room.Mutex.Lock()
	room.Characters[character.Index] = character
	room.Mutex.Unlock()

	s.Mutex.Lock()
	s.CharacterExists[strings.ToLower(charName)] = true // Store the character name in the map
	s.Mutex.Unlock()

	return character, nil
}

func (s *Server) NewCharacter(Name string, Player *Player, Room *Room, archetypeName string) *Character {

	// Generate a new unique index for the character
	characterIndex, err := s.Database.NextIndex("Characters")
	if err != nil {
		log.Printf("Error generating character index: %v", err)
		return nil
	}

	character := &Character{
		Index:      characterIndex,
		Room:       Room,
		Name:       Name,
		Player:     Player,
		Attributes: make(map[string]float64),
		Abilities:  make(map[string]float64),
		Server:     s,
	}

	// Apply archetype attributes and abilities if an archetype is selected
	if archetypeName != "" {
		if archetype, ok := s.Archetypes.Archetypes[archetypeName]; ok {
			character.Attributes = make(map[string]float64)
			for attr, value := range archetype.Attributes {
				character.Attributes[attr] = value
			}
			character.Abilities = make(map[string]float64)
			for ability, value := range archetype.Abilities {
				character.Abilities[ability] = value
			}
		}
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

	if s.Characters == nil {
		s.Mutex.Lock()
		s.Characters = make(map[string]*Character)
		s.Mutex.Unlock()
	}

	s.Mutex.Lock()
	s.Characters[Name] = character
	s.Mutex.Unlock()

	return character
}

// Converts a Character to CharacterData for serialization
func (c *Character) ToData() *CharacterData {
	return &CharacterData{
		Index:      c.Index,
		Name:       c.Name,
		RoomID:     c.Room.RoomID,
		Attributes: c.Attributes,
		Abilities:  c.Abilities,
	}
}

func (s *Server) WriteCharacter(character *Character) error {
	// Convert Character to CharacterData before marshalling
	characterData := character.ToData()

	// Marshal CharacterData instead of Character
	jsonData, err := json.Marshal(characterData)
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
		err = bucket.Put([]byte(indexKey), jsonData)
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

func (s *Server) LoadCharacter(player *Player, characterIndex uint64) (*Character, error) {
	var characterData []byte
	err := s.Database.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("Characters"))
		if bucket == nil {
			return fmt.Errorf("characters bucket not found")
		}
		indexKey := fmt.Sprintf("%d", characterIndex)
		characterData = bucket.Get([]byte(indexKey))
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
		Index:      cd.Index,
		Room:       room,
		Name:       cd.Name,
		Attributes: cd.Attributes,
		Abilities:  cd.Abilities,
		Player:     player,
		Server:     s,
	}

	if s.Characters == nil {
		s.Mutex.Lock()
		s.Characters = make(map[string]*Character)
		s.Mutex.Unlock()
	}

	s.Mutex.Lock()
	s.Characters[cd.Name] = character
	s.Mutex.Unlock()

	log.Printf("Loaded character %s (Index %d) in Room %d", character.Name, character.Index, room.RoomID)

	return character, nil
}

func (c *Character) InputLoop() {
	// Initially execute the look command with no additional tokens
	executeLookCommand(c, []string{}) // Adjusted to include the tokens parameter

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
		verb, tokens, err := validateCommand(strings.TrimSpace(inputLine), commandHandlers)
		if err != nil {
			c.Player.ToPlayer <- err.Error() + "\n\r"
			c.Player.ToPlayer <- c.Player.Prompt
			continue
		}

		// Execute the command
		if executeCommand(c, verb, tokens) {
			// If command execution indicates to exit (or similar action), break the loop
			// Note: Adjust logic as per your executeCommand's design to handle such conditions
			break
		}

		// Log the command execution
		log.Printf("Player %s issued command: %s", c.Player.Name, strings.Join(tokens, " "))

		// Prompt for the next command
		c.Player.ToPlayer <- c.Player.Prompt
	}
}

func (c *Character) Move(direction string) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if c.Room == nil {
		c.Player.ToPlayer <- "You are not in any room to move from.\n\r"
		return
	}

	log.Printf("Player %s is moving %s", c.Name, direction)

	selectedExit, exists := c.Room.Exits[direction]
	if !exists {
		c.Player.ToPlayer <- "You cannot go that way.\n\r"
		return
	}

	newRoom, exists := c.Server.Rooms[selectedExit.TargetRoom]
	if !exists {
		c.Player.ToPlayer <- "The path leads nowhere.\n\r"
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

func (s *Server) LoadCharacterNames() (map[string]bool, error) {

	names := make(map[string]bool)

	err := s.Database.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Characters"))
		if b == nil {
			return fmt.Errorf("characters bucket not found")
		}

		return b.ForEach(func(k, v []byte) error {
			var cd CharacterData
			if err := json.Unmarshal(v, &cd); err != nil {
				log.Printf("Error unmarshalling character data: %v", err)
			}

			names[strings.ToLower(cd.Name)] = true // Store the character name in the map
			return nil
		})
	})

	if len(names) == 0 {
		return nil, fmt.Errorf("no characters found")
	}

	if err != nil {
		return nil, fmt.Errorf("error reading from BoltDB: %w", err)
	}

	return names, nil
}

func (s *Server) SaveActiveCharacters() error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	log.Println("Saving active characters...")

	for _, character := range s.Characters {
		err := s.WriteCharacter(character)
		if err != nil {
			return fmt.Errorf("error saving character %s: %w", character.Name, err)
		}
	}

	log.Println("Active characters saved successfully.")

	return nil
}

func (s *Server) LoadArchetypes() (*ArchetypesData, error) {

	archetypesData := &ArchetypesData{Archetypes: make(map[string]Archetype)}

	err := s.Database.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("Archetypes"))
		if bucket == nil {
			return fmt.Errorf("archetypes bucket does not exist")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var archetype Archetype
			if err := json.Unmarshal(v, &archetype); err != nil {
				return err
			}
			log.Println("Reading", string(k), archetype)
			archetypesData.Archetypes[string(k)] = archetype
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return archetypesData, nil
}
