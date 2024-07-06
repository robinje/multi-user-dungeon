package core

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	bolt "go.etcd.io/bbolt"
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

func SelectCharacter(player *Player, server *Server) (*Character, error) {

	log.Printf("Player %s is selecting a character", player.Name)

	var options []string

	sendCharacterOptions := func() {
		player.ToPlayer <- "Select a character:\n\r"
		player.ToPlayer <- "0: Create a new character\n\r"

		if len(player.CharacterList) > 0 {
			i := 1
			for name := range player.CharacterList {
				player.ToPlayer <- fmt.Sprintf("%d: %s\n\r", i, name)
				options = append(options, name)
				i++
			}
		}
		player.ToPlayer <- "Enter the number of your choice: "
	}

	for {
		sendCharacterOptions()

		input, ok := <-player.FromPlayer
		if !ok {
			return nil, fmt.Errorf("failed to receive input")
		}

		choice, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil || choice < 0 || choice > len(options) {
			player.ToPlayer <- "Invalid choice. Please select a valid option.\n\r"
			continue
		}

		if choice == 0 {
			return CreateCharacter(player, server)
		} else if choice <= len(options) {
			characterName := options[choice-1]
			return server.Database.LoadCharacter(player.CharacterList[characterName], player, server)
		}
	}
}

func CreateCharacter(player *Player, server *Server) (*Character, error) {

	log.Printf("Player %s is creating a new character", player.Name)

	player.ToPlayer <- "\n\rEnter your character name: "

	charName, ok := <-player.FromPlayer
	if !ok {
		return nil, fmt.Errorf("failed to receive character name input")
	}

	charName = strings.TrimSpace(charName)

	if len(charName) == 0 {
		return nil, fmt.Errorf("character name cannot be empty")
	}

	if len(charName) > 15 {
		return nil, fmt.Errorf("character name must be 15 characters or fewer")
	}

	if server.CharacterExists[strings.ToLower(charName)] {
		return nil, fmt.Errorf("character already exists")
	}

	var selectedArchetype string
	if server.Archetypes != nil && len(server.Archetypes.Archetypes) > 0 {
		for {
			selectionMsg := "\n\rSelect a character archetype.\n\r"
			archetypeOptions := make([]string, 0, len(server.Archetypes.Archetypes))
			for name, archetype := range server.Archetypes.Archetypes {
				archetypeOptions = append(archetypeOptions, name+" - "+archetype.Description)
			}
			sort.Strings(archetypeOptions)

			for i, option := range archetypeOptions {
				selectionMsg += fmt.Sprintf("%d: %s\n\r", i+1, option)
			}

			selectionMsg += "Enter the number of your choice: "
			player.ToPlayer <- selectionMsg

			selection, ok := <-player.FromPlayer
			if !ok {
				return nil, fmt.Errorf("failed to receive archetype selection")
			}

			selectionNum, err := strconv.Atoi(strings.TrimSpace(selection))
			if err == nil && selectionNum >= 1 && selectionNum <= len(archetypeOptions) {
				selectedOption := archetypeOptions[selectionNum-1]
				selectedArchetype = strings.Split(selectedOption, " - ")[0]
				break
			} else {
				player.ToPlayer <- "Invalid selection. Please select a valid archetype number."
			}
		}
	}

	log.Printf("Creating character with name: %s", charName)

	room, ok := server.Rooms[1]
	if !ok {
		room, ok = server.Rooms[0]
		if !ok {
			return nil, fmt.Errorf("no starting room found")
		}
	}

	character := server.NewCharacter(charName, player, room, selectedArchetype)

	if room.Characters == nil {
		room.Characters = make(map[uint64]*Character)
	}

	room.Mutex.Lock()
	room.Characters[character.Index] = character
	room.Mutex.Unlock()

	server.Mutex.Lock()
	server.CharacterExists[strings.ToLower(charName)] = true
	server.Mutex.Unlock()

	return character, nil
}

func WearItem(c *Character, item *Item) error {

	log.Printf("Character %s is attempting to wear item %s", c.Name, item.Name)

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if !item.Wearable {
		return fmt.Errorf("this item cannot be worn")
	}

	for _, location := range item.WornOn {
		if !WearLocations[location] {
			return fmt.Errorf("invalid wear location: %s", location)
		}
	}

	for _, location := range item.WornOn {
		if _, exists := c.Inventory[location]; exists {
			return fmt.Errorf("you are already wearing something on your %s", location)
		}
	}

	delete(c.Inventory, item.Name)

	for _, location := range item.WornOn {
		c.Inventory[location] = item
	}

	item.IsWorn = true

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

func RemoveWornItem(c *Character, itemOrLocation interface{}) (*Item, error) {

	log.Printf("Character %s is removing worn item", c.Name)

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	var itemToRemove *Item
	var exists bool

	switch v := itemOrLocation.(type) {
	case string:
		itemToRemove, exists = c.Inventory[v]
		if !exists {
			return nil, fmt.Errorf("you are not wearing anything on your %s", v)
		}
	case *Item:
		if !v.IsWorn {
			return nil, fmt.Errorf("the item %s is not being worn", v.Name)
		}
		itemToRemove = v
	default:
		return nil, fmt.Errorf("invalid argument type for RemoveWornItem")
	}

	for _, loc := range itemToRemove.WornOn {
		if c.Inventory[loc] == itemToRemove {
			delete(c.Inventory, loc)
		}
	}

	if _, exists := c.Inventory[itemToRemove.Name]; !exists {
		c.Inventory[itemToRemove.Name] = itemToRemove
	}

	itemToRemove.IsWorn = false

	return itemToRemove, nil
}
