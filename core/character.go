package core

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

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

func (c *Character) ToData() *CharacterData {
	inventoryIDs := make(map[string]string, len(c.Inventory))
	for name, obj := range c.Inventory {
		inventoryIDs[name] = obj.ID.String()
	}

	return &CharacterData{
		Index:      c.ID.String(),
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
	Index, err := uuid.Parse(cd.Index)
	if err != nil {
		return fmt.Errorf("parse character index: %w", err)
	}
	c.ID = Index
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
	characterData := character.ToData()
	av, err := dynamodbattribute.MarshalMap(characterData)
	if err != nil {
		return fmt.Errorf("error marshalling character data: %w", err)
	}

	key := map[string]*dynamodb.AttributeValue{
		"Index": {S: aws.String(character.ID.String())},
	}

	err = kp.Put("characters", key, av)
	if err != nil {
		return fmt.Errorf("error writing character data: %w", err)
	}

	log.Printf("Successfully wrote character %s with ID %s to database", character.Name, character.ID)
	return nil
}

func (kp *KeyPair) LoadCharacter(characterID uuid.UUID, player *Player, server *Server) (*Character, error) {
	key := map[string]*dynamodb.AttributeValue{
		"Index": {S: aws.String(characterID.String())},
	}

	var cd CharacterData
	err := kp.Get("characters", key, &cd)
	if err != nil {
		return nil, fmt.Errorf("error loading character data: %w", err)
	}

	character := &Character{
		Server: server,
		Player: player,
	}

	if err := character.FromData(&cd); err != nil {
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
		log.Printf("Added character %s (ID: %s) to room %d", character.Name, character.ID, character.Room.RoomID)
	} else {
		log.Printf("Warning: Character %s (ID: %s) loaded without a valid room", character.Name, character.ID)
	}

	log.Printf("Loaded character %s (ID: %s) in Room %d", character.Name, character.ID, character.Room.RoomID)

	return character, nil
}

func (kp *KeyPair) LoadCharacterNames() (map[string]bool, error) {
	names := make(map[string]bool)

	var characters []struct {
		Name string `dynamodbav:"Name"`
	}

	err := kp.Scan("characters", &characters)
	if err != nil {
		return nil, fmt.Errorf("error scanning characters: %w", err)
	}

	for _, character := range characters {
		names[strings.ToLower(character.Name)] = true
	}

	if len(names) == 0 {
		return names, fmt.Errorf("no characters found")
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

	return nil
}

func ListInventory(c *Character) string {

	log.Printf("Character %s is listing inventory", c.Name)

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	var held, worn []string
	wornItems := make(map[string]bool) // To avoid duplicates in worn items list

	for _, item := range c.Inventory {
		if item.IsWorn {
			if !wornItems[item.Name] {
				worn = append(worn, fmt.Sprintf("%s (worn on %s)", item.Name, strings.Join(item.WornOn, ", ")))
				wornItems[item.Name] = true
			}
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

func AddToInventory(c *Character, item *Item) {

	log.Printf("Character %s is adding item %s to inventory", c.Name, item.Name)

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if item.Wearable && len(item.WornOn) > 0 {
		for _, location := range item.WornOn {
			c.Inventory[location] = item
		}
		item.IsWorn = true
	} else {
		c.Inventory[item.Name] = item
	}
}

func FindInInventory(c *Character, itemName string) *Item {

	log.Printf("Character %s is searching inventory for item %s", c.Name, itemName)

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

func RemoveFromInventory(c *Character, item *Item) {

	log.Printf("Character %s is removing item %s from inventory", c.Name, item.Name)

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if item.IsWorn {
		for _, location := range item.WornOn {
			delete(c.Inventory, location)
		}
		item.IsWorn = false
	} else {
		delete(c.Inventory, item.Name)
	}
}

func CanCarryItem(c *Character, item *Item) bool {

	log.Printf("Character %s is checking if they can carry item %s", c.Name, item.Name)

	// Placeholder implementation
	return true
}

func RemoveWornItem(c *Character, item *Item) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if item == nil {
		return fmt.Errorf("no item specified")
	}

	var wornLocation string
	for location, invItem := range c.Inventory {
		if invItem == item && item.IsWorn {
			wornLocation = location
			break
		}
	}

	if wornLocation == "" {
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

	delete(c.Inventory, wornLocation)
	item.IsWorn = false
	c.Inventory[handSlot] = item

	return nil
}
