package core

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	bolt "go.etcd.io/bbolt"
)

func (c *Character) ToData() *CharacterData {
	inventoryIDs := make(map[string]string, len(c.Inventory))
	for name, obj := range c.Inventory {
		inventoryIDs[name] = obj.ID.String()
	}

	return &CharacterData{
		Index:      c.Index,
		PlayerID:   c.Player.PlayerID,
		Name:       c.Name,
		Attributes: c.Attributes,
		Abilities:  c.Abilities,
		Essence:    c.Essence,
		Health:     c.Health,
		RoomID:     c.Room.RoomID,
		Inventory:  inventoryIDs,
	}
}

func (c *Character) FromData(cd *CharacterData) error {
	c.Index = cd.Index
	c.Name = cd.Name
	c.Attributes = cd.Attributes
	c.Abilities = cd.Abilities
	c.Essence = cd.Essence
	c.Health = cd.Health

	room, exists := c.Server.Rooms[cd.RoomID]
	if !exists {
		log.Printf("room with ID %d not found", cd.RoomID)
		room = c.Server.Rooms[0]
	}
	c.Room = room

	c.Inventory = make(map[string]*Item, len(cd.Inventory))
	for key, objID := range cd.Inventory {
		obj, err := c.Server.Database.LoadItem(objID, false)
		if err != nil {
			log.Printf("Error loading object %s for character %s: %v", objID, c.Name, err)
			continue
		}
		c.Inventory[key] = obj
	}

	return nil
}

func (s *Server) NewCharacter(name string, player *Player, room *Room, archetypeName string) *Character {
	characterIndex, err := s.Database.NextIndex("Characters")
	if err != nil {
		log.Printf("Error generating character index: %v", err)
		return nil
	}

	character := &Character{
		Index:      characterIndex,
		Room:       room,
		Name:       name,
		Player:     player,
		Health:     float64(s.Health),
		Essence:    float64(s.Essence),
		Attributes: make(map[string]float64),
		Abilities:  make(map[string]float64),
		Inventory:  make(map[string]*Item),
		Server:     s,
	}

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

	return character
}

// WriteCharacter persists a character to the database.
func (kp *KeyPair) WriteCharacter(character *Character) error {
	character.Mutex.Lock()
	defer character.Mutex.Unlock()

	characterData := character.ToData()
	jsonData, err := json.Marshal(characterData)
	if err != nil {
		return fmt.Errorf("marshal character data: %w", err)
	}

	return kp.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("Characters"))
		if err != nil {
			return fmt.Errorf("create characters bucket: %w", err)
		}

		indexKey := strconv.FormatUint(character.Index, 10)
		if err := bucket.Put([]byte(indexKey), jsonData); err != nil {
			return fmt.Errorf("write character data: %w", err)
		}

		log.Printf("Successfully wrote character %s with Index %d to database", character.Name, character.Index)
		return nil
	})
}

func (kp *KeyPair) LoadCharacter(characterIndex uint64, player *Player, server *Server) (*Character, error) {
	var characterData []byte
	err := kp.db.View(func(tx *bolt.Tx) error {
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
		Server: server,
		Player: player,
	}

	if err := character.FromData(&cd); err != nil {
		return nil, fmt.Errorf("error loading character from data: %w", err)
	}

	log.Printf("Loaded character %s (Index %d) in Room %d", character.Name, character.Index, character.Room.RoomID)

	return character, nil
}

func (kp *KeyPair) LoadCharacterNames() (map[string]bool, error) {
	names := make(map[string]bool)

	err := kp.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Characters"))
		if b == nil {
			return fmt.Errorf("characters bucket not found")
		}

		return b.ForEach(func(k, v []byte) error {
			var cd CharacterData
			if err := json.Unmarshal(v, &cd); err != nil {
				log.Printf("Error unmarshalling character data: %v", err)
			}

			names[strings.ToLower(cd.Name)] = true
			return nil
		})
	})

	if len(names) == 0 {
		return names, fmt.Errorf("no characters found")
	}

	if err != nil {
		return names, fmt.Errorf("error reading from BoltDB: %w", err)
	}

	return names, nil
}

func SaveActiveCharacters(s *Server) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	log.Println("Saving active characters...")

	for _, character := range s.Characters {
		err := s.Database.WriteCharacter(character)
		if err != nil {
			return fmt.Errorf("error saving character %s: %w", character.Name, err)
		}
	}

	log.Println("Active characters saved successfully.")

	return nil
}
