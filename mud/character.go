package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Character struct {
	Index      uint64
	Player     *Player
	Name       string
	Attributes map[string]float64
	Abilities  map[string]float64
	Essence    float64
	Health     float64
	Room       *Room
	Inventory  []*Object
	RightHand  *Object
	LeftHand   *Object
	Server     *Server
	Mutex      sync.Mutex
}

// CharacterData for unmarshalling character.
type CharacterData struct {
	Index        uint64             `json:"index"`
	PlayerID     uint64             `json:"playerID"`
	Name         string             `json:"name"`
	Attributes   map[string]float64 `json:"attributes"`
	Abilities    map[string]float64 `json:"abilities"`
	Essence      float64            `json:"essence"`
	Health       float64            `json:"health"`
	RoomID       int64              `json:"roomID"`
	InventoryIDs []uint64           `json:"inventoryIDs"`
	RightHandID  uint64             `json:"rightHandID"`
	LeftHandID   uint64             `json:"leftHandID"`
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

func (s *Server) AutoSaveCharacters() {
	for {
		// Sleep for the configured duration
		time.Sleep(time.Duration(s.AutoSave) * time.Minute)

		// Save the characters to the database
		if err := s.SaveActiveCharacters(); err != nil {
			log.Printf("Failed to save characters: %v", err)
		}
	}
}

// Converts a Character to CharacterData for serialization
func (c *Character) ToData() *CharacterData {
	inventoryIDs := make([]uint64, len(c.Inventory))
	for i, obj := range c.Inventory {
		inventoryIDs[i] = obj.Index
	}

	var rightHandID, leftHandID uint64
	if c.RightHand != nil {
		rightHandID = c.RightHand.Index
	}
	if c.LeftHand != nil {
		leftHandID = c.LeftHand.Index
	}

	return &CharacterData{
		Index:        c.Index,
		PlayerID:     c.Player.Index,
		Name:         c.Name,
		Attributes:   c.Attributes,
		Abilities:    c.Abilities,
		Essence:      c.Essence,
		Health:       c.Health,
		RoomID:       c.Room.RoomID,
		InventoryIDs: inventoryIDs,
		RightHandID:  rightHandID,
		LeftHandID:   leftHandID,
	}
}

func (c *Character) FromData(cd *CharacterData) error {
	c.Index = cd.Index
	c.Name = cd.Name
	c.Attributes = cd.Attributes
	c.Abilities = cd.Abilities
	c.Essence = cd.Essence
	c.Health = cd.Health

	// Load the room
	room, exists := c.Server.Rooms[cd.RoomID]
	if !exists {
		return fmt.Errorf("room with ID %d not found", cd.RoomID)
	}
	c.Room = room

	// Load objects from the character's inventory
	c.Inventory = make([]*Object, len(cd.InventoryIDs))
	for i, objIndex := range cd.InventoryIDs {
		obj, err := c.Server.Database.LoadObject(objIndex, false)
		if err != nil {
			log.Printf("Error loading object %d for character %s: %v", objIndex, c.Name, err)
			continue
		}
		c.Inventory[i] = obj
	}

	// Load right hand object
	if cd.RightHandID != 0 {
		rightHand, err := c.Server.Database.LoadObject(cd.RightHandID, false)
		if err != nil {
			log.Printf("Error loading right hand object %d for character %s: %v", cd.RightHandID, c.Name, err)
		} else {
			c.RightHand = rightHand
		}
	}

	// Load left hand object
	if cd.LeftHandID != 0 {
		leftHand, err := c.Server.Database.LoadObject(cd.LeftHandID, false)
		if err != nil {
			log.Printf("Error loading left hand object %d for character %s: %v", cd.LeftHandID, c.Name, err)
		} else {
			c.LeftHand = leftHand
		}
	}

	return nil
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
		Health:     float64(s.Health),
		Essence:    float64(s.Essence),
		Attributes: make(map[string]float64),
		Abilities:  make(map[string]float64),
		Inventory:  make([]*Object, 0),
		RightHand:  nil,
		LeftHand:   nil,
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

	character := &Character{
		Server: s,
		Player: player,
	}

	if err := character.FromData(&cd); err != nil {
		return nil, fmt.Errorf("error loading character from data: %w", err)
	}

	if s.Characters == nil {
		s.Mutex.Lock()
		s.Characters = make(map[string]*Character)
		s.Mutex.Unlock()
	}

	s.Mutex.Lock()
	s.Characters[cd.Name] = character
	s.Mutex.Unlock()

	log.Printf("Loaded character %s (Index %d) in Room %d", character.Name, character.Index, character.Room.RoomID)

	return character, nil
}

func (k *KeyPair) LoadCharacterNames() (map[string]bool, error) {
	names := make(map[string]bool)

	err := k.db.View(func(tx *bolt.Tx) error {
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
