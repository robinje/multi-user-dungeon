package main

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/robinje/multi-user-dungeon/core"
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

func SelectCharacter(player *core.Player, server *core.Server) (*core.Character, error) {
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

func CreateCharacter(player *core.Player, server *core.Server) (*core.Character, error) {
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
		room.Characters = make(map[uint64]*core.Character)
	}

	room.Mutex.Lock()
	room.Characters[character.Index] = character
	room.Mutex.Unlock()

	server.Mutex.Lock()
	server.CharacterExists[strings.ToLower(charName)] = true
	server.Mutex.Unlock()

	return character, nil
}

func WearItem(c *core.Character, item *core.Item) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if !item.Wearable {
		return fmt.Errorf("this item cannot be worn")
	}

	for _, location := range item.WornOn {
		if WearLocations[location] == false {
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

func ListInventory(c *core.Character) string {
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

func AddToInventory(c *core.Character, item *core.Item) {
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

func FindInInventory(c *core.Character, itemName string) *core.Item {
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

func RemoveFromInventory(c *core.Character, item *core.Item) {
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

func CanCarryItem(c *core.Character, item *core.Item) bool {
	// Placeholder implementation
	return true
}

func RemoveWornItem(c *core.Character, itemOrLocation interface{}) (*core.Item, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	var itemToRemove *core.Item
	var exists bool

	switch v := itemOrLocation.(type) {
	case string:
		itemToRemove, exists = c.Inventory[v]
		if !exists {
			return nil, fmt.Errorf("you are not wearing anything on your %s", v)
		}
	case *core.Item:
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
