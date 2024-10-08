package core

import (
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/bits-and-blooms/bloom/v3"
	"github.com/google/uuid"
)

const FalsePositiveRate = 0.01 // 1% false positive rate

// WearLocations defines all possible locations where an item can be worn
var WearLocations = map[string]bool{
	"head":         true,
	"neck":         true,
	"shoulders":    true,
	"chest":        true,
	"back":         true,
	"arms":         true,
	"hands":        true,
	"waist":        true,
	"legs":         true,
	"feet":         true,
	"left_finger":  true,
	"right_finger": true,
	"left_wrist":   true,
	"right_wrist":  true,
}

// ToData converts a Character object into a CharacterData struct for database storage.
func (c *Character) ToData() *CharacterData {
	inventoryIDs := make(map[string]string)
	for name, item := range c.Inventory {
		inventoryIDs[name] = item.ID.String()
	}

	return &CharacterData{
		CharacterID:   c.ID.String(),
		PlayerID:      c.Player.PlayerID,
		CharacterName: c.Name,
		Attributes:    c.Attributes,
		Abilities:     c.Abilities,
		Essence:       c.Essence,
		Health:        c.Health,
		RoomID:        c.Room.RoomID,
		Inventory:     inventoryIDs,
	}
}

// FromData populates a Character object from a CharacterData struct retrieved from the database.
func (c *Character) FromData(cd *CharacterData, server *Server) error {
	var err error
	c.ID, err = uuid.Parse(cd.CharacterID)
	if err != nil {
		return fmt.Errorf("parse character ID: %w", err)
	}
	c.Name = cd.CharacterName
	c.Attributes = cd.Attributes
	c.Abilities = cd.Abilities
	c.Essence = cd.Essence
	c.Health = cd.Health

	// Retrieve the room; if not found, default to room ID 0
	room, exists := server.Rooms[cd.RoomID]
	if !exists {
		Logger.Warn("Room not found, defaulting to room ID 0", "roomID", cd.RoomID)
		room, exists = server.Rooms[0]
		if !exists {
			return fmt.Errorf("default room not found")
		}
	}
	c.Room = room
	c.Server = server

	// Initialize inventory
	c.Inventory = make(map[string]*Item)
	for name, itemIDStr := range cd.Inventory {
		itemID, err := uuid.Parse(itemIDStr)
		if err != nil {
			Logger.Error("Error parsing item UUID", "itemID", itemIDStr, "error", err)
			continue
		}
		item, err := server.Database.LoadItem(itemID.String())
		if err != nil {
			Logger.Error("Error loading item for character", "itemID", itemID, "characterName", c.Name, "error", err)
			continue
		}
		c.Inventory[name] = item
	}

	return nil
}

// NewCharacter creates a new character with the specified name and archetype.
func (s *Server) NewCharacter(name string, player *Player, room *Room, archetypeName string) *Character {
	character := &Character{
		ID:         uuid.New(),
		Room:       room,
		Name:       name,
		Player:     player,
		Health:     float64(s.Health),
		Essence:    float64(s.Essence),
		Attributes: make(map[string]float64),
		Abilities:  make(map[string]float64),
		Inventory:  make(map[string]*Item),
		Server:     s,
		Mutex:      sync.Mutex{},
	}

	// Add character name to bloom filter to prevent duplicates
	s.AddCharacterName(name)

	// Apply archetype attributes and abilities
	if archetypeName != "" {
		if archetype, ok := s.Archetypes.Archetypes[archetypeName]; ok {
			for attr, value := range archetype.Attributes {
				character.Attributes[attr] = value
			}
			for ability, value := range archetype.Abilities {
				character.Abilities[ability] = value
			}
		}
	}

	return character
}

// WriteCharacter saves the character to the DynamoDB database.
func (kp *KeyPair) WriteCharacter(character *Character) error {

	characterData := character.ToData()

	err := kp.Put("characters", characterData)
	if err != nil {
		Logger.Error("Error writing character data", "characterName", character.Name, "error", err)
		return fmt.Errorf("error writing character data: %w", err)
	}

	Logger.Info("Successfully wrote character to database", "characterName", character.Name, "characterID", character.ID)
	return nil
}

// LoadCharacter retrieves a character from the DynamoDB database and reconstructs the Character object.
func (kp *KeyPair) LoadCharacter(characterID uuid.UUID, player *Player, server *Server) (*Character, error) {
	key := map[string]*dynamodb.AttributeValue{
		"CharacterID": {S: aws.String(characterID.String())},
	}

	var cd CharacterData
	err := kp.Get("characters", key, &cd)
	if err != nil {
		Logger.Error("Error loading character data", "characterID", characterID, "error", err)
		return nil, fmt.Errorf("error loading character data: %w", err)
	}

	character := &Character{
		Server: server,
		Player: player,
		Mutex:  sync.Mutex{},
	}

	if err := character.FromData(&cd, server); err != nil {
		Logger.Error("Error reconstructing character from data", "characterID", characterID, "error", err)
		return nil, fmt.Errorf("error loading character from data: %w", err)
	}

	// Ensure the character is added to the room's character list
	if character.Room != nil {
		character.Room.Mutex.Lock()
		if character.Room.Characters == nil {
			character.Room.Characters = make(map[uuid.UUID]*Character)
		}
		character.Room.Characters[character.ID] = character
		character.Room.Mutex.Unlock()
		Logger.Info("Added character to room", "characterName", character.Name, "characterID", character.ID, "roomID", character.Room.RoomID)
	} else {
		Logger.Warn("Character loaded without a valid room", "characterName", character.Name, "characterID", character.ID)
	}

	Logger.Info("Loaded character", "characterName", character.Name, "characterID", character.ID)
	return character, nil
}

// DeleteCharacter removes a character from the player's character list and the database.
func (s *Server) DeleteCharacter(player *Player, characterName string) error {
	Logger.Info("Attempting to delete character", "playerName", player.PlayerID, "characterName", characterName)

	// Check if the character exists in the player's character list
	characterID, exists := player.CharacterList[characterName]
	if !exists {
		return fmt.Errorf("character %s not found for player %s", characterName, player.PlayerID)
	}

	// Remove the character from the player's character list
	delete(player.CharacterList, characterName)

	// Update the player data in the database
	err := s.Database.WritePlayer(player)
	if err != nil {
		Logger.Error("Failed to update player data after character deletion", "playerName", player.PlayerID, "error", err)
		return fmt.Errorf("failed to update player data: %w", err)
	}

	// Delete the character from the database
	key := map[string]*dynamodb.AttributeValue{
		"CharacterID": {S: aws.String(characterID.String())},
	}
	err = s.Database.Delete("characters", key)
	if err != nil {
		Logger.Error("Failed to delete character from database", "characterName", characterName, "characterID", characterID, "error", err)
		return fmt.Errorf("failed to delete character from database: %w", err)
	}

	Logger.Info("Successfully deleted character", "playerName", player.PlayerID, "characterName", characterName, "characterID", characterID)
	return nil
}

// LoadCharacterNames loads all character names from the database to initialize the bloom filter.
func (kp *KeyPair) LoadCharacterNames() (map[string]bool, error) {
	names := make(map[string]bool)

	var characters []struct {
		CharacterName string `dynamodbav:"Name"`
	}

	err := kp.Scan("characters", &characters)
	if err != nil {
		Logger.Error("Error scanning characters table", "error", err)
		return nil, fmt.Errorf("error scanning characters: %w", err)
	}

	for _, character := range characters {
		names[strings.ToLower(character.CharacterName)] = true
	}

	if len(names) == 0 {
		Logger.Warn("No characters found in the database")
		return names, nil // Return empty map without error
	}

	return names, nil
}

// InitializeBloomFilter initializes the bloom filter with existing character names.
func (server *Server) InitializeBloomFilter() error {
	characterNames, err := server.Database.LoadCharacterNames()
	if err != nil {
		return fmt.Errorf("failed to load character names: %w", err)
	}

	n := uint(max(len(characterNames), 100)) // Use at least 100 as the initial size
	fpRate := FalsePositiveRate

	server.CharacterBloomFilter = bloom.NewWithEstimates(n, fpRate)

	for name := range characterNames {
		server.CharacterBloomFilter.AddString(strings.ToLower(name))
	}

	Logger.Info("Bloom filter initialized", "estimatedSize", n, "falsePositiveRate", fpRate)
	return nil
}

// AddCharacterName adds a character name to the bloom filter to prevent duplicates.
func (server *Server) AddCharacterName(name string) {
	server.Mutex.Lock()
	defer server.Mutex.Unlock()

	server.CharacterBloomFilter.AddString(strings.ToLower(name))
	Logger.Info("Added character name to bloom filter", "characterName", name)
}

// CharacterNameExists checks if a character name already exists using the bloom filter.
func (server *Server) CharacterNameExists(name string) bool {
	exists := server.CharacterBloomFilter.TestString(strings.ToLower(name))
	if exists {
		Logger.Info("Character name exists", "characterName", name)
	}
	return exists
}

// SaveActiveCharacters saves all active characters to the database.
func SaveActiveCharacters(s *Server) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	Logger.Info("Saving active characters...")

	for _, character := range s.Characters {
		err := s.Database.WriteCharacter(character)
		if err != nil {
			Logger.Error("Error saving character", "characterName", character.Name, "error", err)
			continue // Continue saving other characters
		}
	}

	Logger.Info("Active characters saved successfully.")
	return nil
}

// WearItem allows a character to wear an item from their inventory.
func WearItem(c *Character, item *Item) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	// Check if the item is in a hand slot
	inHand := false
	var handSlot string
	for slot, handItem := range c.Inventory {
		if (slot == "left_hand" || slot == "right_hand") && handItem == item {
			inHand = true
			handSlot = slot
			break
		}
	}

	if !inHand {
		return fmt.Errorf("you need to be holding the item to wear it")
	}

	if !item.Wearable {
		return fmt.Errorf("this item cannot be worn")
	}

	for _, location := range item.WornOn {
		if !WearLocations[location] {
			return fmt.Errorf("invalid wear location: %s", location)
		}
		if c.Inventory[location] != nil {
			return fmt.Errorf("you are already wearing something on your %s", location)
		}
	}

	for _, location := range item.WornOn {
		c.Inventory[location] = item
	}

	item.IsWorn = true
	delete(c.Inventory, handSlot) // Remove from hand slot

	Logger.Info("Item worn", "characterName", c.Name, "itemName", item.Name, "wornOn", item.WornOn)
	return nil
}

// ListInventory lists the items in a character's inventory.
func ListInventory(c *Character) string {
	Logger.Debug("Character is listing inventory", "characterName", c.Name)

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	var held, worn []string
	wornItems := make(map[string]bool) // To avoid duplicates in worn items list

	for slot, item := range c.Inventory {
		if item.IsWorn {
			if !wornItems[item.Name] {
				worn = append(worn, fmt.Sprintf("%s (worn on %s)", item.Name, strings.Join(item.WornOn, ", ")))
				wornItems[item.Name] = true
			}
		} else if slot == "left_hand" || slot == "right_hand" {
			held = append(held, fmt.Sprintf("%s (in %s)", item.Name, slot))
		} else {
			held = append(held, item.Name)
		}
	}

	result := "\n\rInventory:\n\r"
	if len(held) > 0 {
		result += "Held items: " + strings.Join(held, ", ") + "\n\r"
	}
	if len(worn) > 0 {
		result += "Worn items: " + strings.Join(worn, ", ") + "\n\r"
	}
	if len(held) == 0 && len(worn) == 0 {
		result += "Your inventory is empty.\n\r"
	}

	return result
}

// AddToInventory adds an item to the character's inventory.
func AddToInventory(c *Character, item *Item) {
	Logger.Debug("Character is adding item to inventory", "characterName", c.Name, "itemName", item.Name)

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if item.Wearable && len(item.WornOn) > 0 {
		for _, location := range item.WornOn {
			c.Inventory[location] = item
		}
		item.IsWorn = true
	} else {
		// Place in the first available hand slot
		if c.Inventory["right_hand"] == nil {
			c.Inventory["right_hand"] = item
		} else if c.Inventory["left_hand"] == nil {
			c.Inventory["left_hand"] = item
		} else {
			// If both hands are full, add to general inventory
			c.Inventory[item.Name] = item
		}
	}

	Logger.Info("Item added to inventory", "characterName", c.Name, "itemName", item.Name)
}

// FindInInventory searches for an item in the character's inventory by name.
func FindInInventory(c *Character, itemName string) *Item {
	Logger.Debug("Character is searching inventory for item", "characterName", c.Name, "itemName", itemName)

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	lowercaseName := strings.ToLower(itemName)

	for _, item := range c.Inventory {
		if strings.Contains(strings.ToLower(item.Name), lowercaseName) {
			return item
		}
	}

	return nil
}

// RemoveFromInventory removes an item from the character's inventory.
func RemoveFromInventory(c *Character, item *Item) {
	Logger.Debug("Character is removing item from inventory", "characterName", c.Name, "itemName", item.Name)

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if item.IsWorn {
		for _, location := range item.WornOn {
			delete(c.Inventory, location)
		}
		item.IsWorn = false
	} else {
		// Remove from hand slots or general inventory
		for slot, invItem := range c.Inventory {
			if invItem == item {
				delete(c.Inventory, slot)
				break
			}
		}
	}

	Logger.Info("Item removed from inventory", "characterName", c.Name, "itemName", item.Name)
}

// CanCarryItem checks if the character can carry the specified item.
// This is a placeholder for future weight and capacity checks.
func CanCarryItem(c *Character, item *Item) bool {
	Logger.Info("Character is checking if they can carry item", "characterName", c.Name, "itemName", item.Name)

	// Placeholder implementation; always returns true for now
	return true
}

// RemoveWornItem allows a character to remove a worn item.
func RemoveWornItem(c *Character, item *Item) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if item == nil {
		return fmt.Errorf("no item specified")
	}

	// Check if the item is worn
	isWorn := false
	for _, invItem := range c.Inventory {
		if invItem == item && item.IsWorn {
			isWorn = true
			break
		}
	}

	if !isWorn {
		return fmt.Errorf("you are not wearing that item")
	}

	// Try to place the item in the right hand first, then the left hand if right is occupied
	var handSlot string
	if c.Inventory["right_hand"] == nil {
		handSlot = "right_hand"
	} else if c.Inventory["left_hand"] == nil {
		handSlot = "left_hand"
	}

	if handSlot == "" {
		return fmt.Errorf("your hands are full. You need a free hand to remove an item")
	}

	// Remove item from worn locations
	for _, location := range item.WornOn {
		delete(c.Inventory, location)
	}
	item.IsWorn = false

	// Place item in hand slot
	c.Inventory[handSlot] = item

	Logger.Info("Item removed from worn location and placed in hand", "characterName", c.Name, "itemName", item.Name, "handSlot", handSlot)
	return nil
}

// getOtherCharacters returns a list of character names in the room, excluding the current character.
func getOtherCharacters(r *Room, currentCharacter *Character) []string {
	if r == nil || r.Characters == nil {
		Logger.Warn("Room or Characters map is nil in getOtherCharacters")
		return []string{}
	}

	otherCharacters := make([]string, 0)
	for _, c := range r.Characters {
		if c != nil && c != currentCharacter {
			otherCharacters = append(otherCharacters, c.Name)
		}
	}

	Logger.Info("Found other characters in room", "count", len(otherCharacters), "room_id", r.RoomID)
	return otherCharacters
}
