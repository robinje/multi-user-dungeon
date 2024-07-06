package core

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
)

type CommandHandler func(character *Character, tokens []string) bool

var CommandHandlers = map[string]CommandHandler{
	"quit":      ExecuteQuitCommand,
	"show":      ExecuteShowCommand,
	"look":      ExecuteLookCommand,
	"say":       ExecuteSayCommand,
	"go":        ExecuteGoCommand,
	"help":      ExecuteHelpCommand,
	"who":       ExecuteWhoCommand,
	"password":  ExecutePasswordCommand,
	"challenge": ExecuteChallengeCommand,
	"take":      ExecuteTakeCommand, // Add the new take command
	"get":       ExecuteTakeCommand, // Alias for take command
	"drop":      ExecuteDropCommand,
	"inventory": ExecuteInventoryCommand,
	"wear":      ExecuteWearCommand,
	"remove":    ExecuteRemoveCommand,
	"examine":   ExecuteExamineCommand,
	"i":         ExecuteInventoryCommand, // Alias for inventory command
	"inv":       ExecuteInventoryCommand, // Alias for inventory command
	"\"":        ExecuteSayCommand,       // Allow for double quotes to be used as a shortcut for the say command
	"'":         ExecuteSayCommand,       // Allow for single quotes to be used as a shortcut for the say command
	"q!":        ExecuteQuitCommand,      // Allow for q! to be used as a shortcut for the quit command
}

func ValidateCommand(command string) (string, []string, error) {

	log.Printf("Received command: %s", command)

	trimmedCommand := strings.TrimSpace(command)
	tokens := strings.Fields(trimmedCommand)

	if len(tokens) == 0 {
		return "", nil, errors.New("\n\rNo command entered.\n\r")
	}

	verb := strings.ToLower(tokens[0])
	if _, exists := CommandHandlers[verb]; !exists {
		return "", tokens, fmt.Errorf(" command not understood")
	}

	return verb, tokens, nil
}

func ExecuteCommand(character *Character, verb string, tokens []string) bool {

	log.Printf("Executing command: %s", verb)

	handler, ok := CommandHandlers[verb]
	if !ok {
		character.Player.ToPlayer <- "\n\rCommand not yet implemented or recognized.\n\r"
		return false
	}
	return handler(character, tokens)
}

func ExecuteQuitCommand(character *Character, tokens []string) bool {
	log.Printf("Player %s is quitting", character.Player.Name)
	character.Player.ToPlayer <- "\n\rGoodbye!"
	SendRoomMessage(character.Room, fmt.Sprintf("\n\r%s has left the game.\n\r", character.Name))

	return true // Indicate that the loop should be exited
}

func ExecuteSayCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is saying something", character.Player.Name)

	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rWhat do you want to say?\n\r"
		return false
	}

	message := strings.Join(tokens[1:], " ")
	broadcastMessage := fmt.Sprintf("\n\r%s says %s\n\r", character.Name, message)

	character.Room.Mutex.Lock()
	for _, c := range character.Room.Characters {
		if c != character {
			// Send message to other characters in the room
			c.Player.ToPlayer <- broadcastMessage
			c.Player.ToPlayer <- c.Player.Prompt
		}
	}
	character.Room.Mutex.Unlock()

	// Send only the broadcast message to the player who issued the command
	character.Player.ToPlayer <- fmt.Sprintf("\n\rYou say %s\n\r", message)

	return false
}

func ExecuteLookCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is looking around", character.Player.Name)

	room := character.Room
	character.Player.ToPlayer <- RoomInfo(room, character)
	return false
}

func ExecuteGoCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is attempting to move", character.Player.Name)

	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rWhich direction do you want to go?\n\r"
		return false
	}

	direction := tokens[1]
	Move(character, direction)
	return false
}

func ExecuteChallengeCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is attempting a challenge", character.Player.Name)

	// Ensure the correct number of arguments are provided
	if len(tokens) < 3 {
		character.Player.ToPlayer <- "\n\rUsage: challenge <attackerScore> <defenderScore>\n\r"
		return false
	}

	// Parse attacker and defender scores from the command arguments
	attackerScore, err := strconv.ParseFloat(tokens[1], 64)
	if err != nil {
		character.Player.ToPlayer <- "\n\rInvalid attacker score format. Please enter a valid number.\n\r"
		return false
	}

	defenderScore, err := strconv.ParseFloat(tokens[2], 64)
	if err != nil {
		character.Player.ToPlayer <- "\n\rInvalid defender score format. Please enter a valid number.\n\r"
		return false
	}

	// Calculate the outcome using the Challenge function
	outcome := Challenge(attackerScore, defenderScore, character.Server.Balance)

	// Provide feedback to the player based on the challenge outcome
	feedbackMessage := fmt.Sprintf("\n\rChallenge outcome: %f\n\r", outcome)
	character.Player.ToPlayer <- feedbackMessage

	return false
}

func ExecuteWhoCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is listing all characters online", character.Player.Name)

	// Retrieve the server instance from the character
	server := character.Server

	characterNames := make([]string, 0, len(server.Characters))
	for name := range server.Characters {
		characterNames = append(characterNames, name)
	}

	// Sort character names for consistent display
	sort.Strings(characterNames)

	// Calculate the number of columns and rows based on console dimensions
	maxNameLength := 15
	columnWidth := maxNameLength + 2 // Adding 2 for spacing between names
	columns := character.Player.ConsoleWidth / columnWidth
	if columns == 0 {
		columns = 1 // Ensure at least one column if console width is too small
	}
	rows := len(characterNames) / columns
	if len(characterNames)%columns != 0 {
		rows++ // Add an extra row for any remainder
	}

	// Prepare message builder to construct the output
	var messageBuilder strings.Builder
	messageBuilder.WriteString("\n\rOnline Characters:\n\r")

	// Loop through rows and columns to construct the output
	for row := 0; row < rows; row++ {
		for col := 0; col < columns; col++ {
			index := row + col*rows
			if index < len(characterNames) {
				messageBuilder.WriteString(fmt.Sprintf("%-15s  ", characterNames[index]))
			}
		}
		messageBuilder.WriteString("\n\r") // New line at the end of each row
	}

	// Send the constructed message to the player
	character.Player.ToPlayer <- messageBuilder.String()

	return false
}

func ExecutePasswordCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is attempting to change their password", character.Player.Name)

	if len(tokens) != 3 {
		character.Player.ToPlayer <- "\n\rUsage: password <oldPassword> <newPassword>\n\r"
		return false
	}

	oldPassword := tokens[1]
	newPassword := tokens[2]

	err := ChangePassword(character.Server, character.Player.Name, oldPassword, newPassword)
	if err != nil {
		log.Printf("Failed to change password for user %s: %v", character.Player.Name, err)
		character.Player.ToPlayer <- "\n\rFailed to change password. Please try again.\n\r"
		return false
	}

	character.Player.ToPlayer <- "\n\rPassword changed successfully.\n\r"
	return false // Keep the command loop running
}

func ExecuteShowCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is displaying character information", character.Player.Name)

	player := character.Player
	var output strings.Builder

	// First row: Character's Name
	output.WriteString(fmt.Sprintf("Name: %s\r\n", character.Name))

	// Health and Essence (integer component only)
	output.WriteString(fmt.Sprintf("Health: %d, Essence: %d\r\n", int(character.Health), int(character.Essence)))

	// Attributes
	output.WriteString("Attributes:\r\n")
	for attr, value := range character.Attributes {
		output.WriteString(fmt.Sprintf("%-15s: %2d\r\n", attr, int(value)))
	}

	// Abilities (only those with scores of 1 or greater)
	output.WriteString("Abilities:\r\n")
	for ability, score := range character.Abilities {
		if score >= 1 {
			output.WriteString(fmt.Sprintf("%-15s: %2d\r\n", ability, int(score)))
		}
	}

	// Send the composed information to the player
	player.ToPlayer <- output.String()

	return false // Keep the command loop running
}

func ExecuteTakeCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is attempting to take an item", character.Player.Name)

	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rUsage: take <item name>\n\r"
		return false
	}

	itemName := strings.ToLower(strings.Join(tokens[1:], " "))
	var itemToTake *Item

	for _, item := range character.Room.Items {
		if strings.Contains(strings.ToLower(item.Name), itemName) && item.CanPickUp {
			itemToTake = item
			break
		}
	}

	if itemToTake == nil {
		character.Player.ToPlayer <- "\n\rYou can't find that item or it can't be picked up.\n\r"
		return false
	}

	if !CanCarryItem(character, itemToTake) {
		character.Player.ToPlayer <- "\n\rYou can't carry any more items.\n\r"
		return false
	}

	character.Room.RemoveItem(itemToTake)
	AddToInventory(character, itemToTake)

	SendRoomMessage(character.Room, fmt.Sprintf("\n\r%s picks up %s.\n\r", character.Name, itemToTake.Name))
	character.Player.ToPlayer <- fmt.Sprintf("\n\rYou take %s.\n\r", itemToTake.Name)
	return false
}

func ExecuteInventoryCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is checking their inventory", character.Player.Name)

	inventoryList := ListInventory(character)
	character.Player.ToPlayer <- inventoryList
	return false
}

func ExecuteDropCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is attempting to drop an item", character.Player.Name)

	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rUsage: drop <item name>\n\r"
		return false
	}

	itemName := strings.ToLower(strings.Join(tokens[1:], " "))
	itemToDrop := FindInInventory(character, itemName)

	if itemToDrop == nil {
		character.Player.ToPlayer <- "\n\rYou don't have that item.\n\r"
		return false
	}

	RemoveFromInventory(character, itemToDrop)
	character.Room.AddItem(itemToDrop)

	character.Player.ToPlayer <- fmt.Sprintf("\n\rYou drop %s.\n\r", itemToDrop.Name)
	SendRoomMessage(character.Room, fmt.Sprintf("\n\r%s drops %s.\n\r", character.Name, itemToDrop.Name))
	return false
}

func ExecuteWearCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is attempting to wear an item", character.Player.Name)

	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rUsage: wear <item name>\n\r"
		return false
	}

	itemName := strings.ToLower(strings.Join(tokens[1:], " "))
	itemToWear := FindInInventory(character, itemName)

	if itemToWear == nil {
		character.Player.ToPlayer <- "\n\rYou don't have that item.\n\r"
		return false
	}

	if !itemToWear.Wearable {
		character.Player.ToPlayer <- "\n\rYou can't wear that.\n\r"
		return false
	}

	if itemToWear.IsWorn {
		character.Player.ToPlayer <- "\n\rYou're already wearing that.\n\r"
		return false
	}

	if err := WearItem(character, itemToWear); err != nil {
		character.Player.ToPlayer <- fmt.Sprintf("\n\r%s\n\r", err.Error())
		return false
	}

	character.Player.ToPlayer <- fmt.Sprintf("\n\rYou wear %s.\n\r", itemToWear.Name)
	SendRoomMessage(character.Room, fmt.Sprintf("\n\r%s wears %s.\n\r", character.Name, itemToWear.Name))
	return false
}

func ExecuteRemoveCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is attempting to remove an item", character.Player.Name)

	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rUsage: remove <item name>\n\r"
		return false
	}

	itemName := strings.ToLower(strings.Join(tokens[1:], " "))
	itemToRemove := FindInInventory(character, itemName)

	if itemToRemove == nil {
		character.Player.ToPlayer <- "\n\rYou don't have that item.\n\r"
		return false
	}

	if !itemToRemove.IsWorn {
		character.Player.ToPlayer <- "\n\rYou're not wearing that.\n\r"
		return false
	}

	removedItem, err := RemoveWornItem(character, itemToRemove)
	if err != nil {
		character.Player.ToPlayer <- fmt.Sprintf("\n\r%s\n\r", err.Error())
		return false
	}

	character.Player.ToPlayer <- fmt.Sprintf("\n\rYou remove %s.\n\r", removedItem.Name)
	SendRoomMessage(character.Room, fmt.Sprintf("\n\r%s removes %s.\n\r", character.Name, removedItem.Name))
	return false
}

func ExecuteExamineCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is examining an item", character.Player.Name)

	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rUsage: examine <item name>\n\r"
		return false
	}

	itemName := strings.ToLower(strings.Join(tokens[1:], " "))

	// Check inventory first
	item := FindInInventory(character, itemName)

	// If not in inventory, check room
	if item == nil {
		for _, roomItem := range character.Room.Items {
			if strings.Contains(strings.ToLower(roomItem.Name), itemName) {
				item = roomItem
				break
			}
		}
	}

	if item == nil {
		character.Player.ToPlayer <- "\n\rYou don't see that item here.\n\r"
		return false
	}

	description := fmt.Sprintf("\n\rItem: %s (ID: %s)\n\r", item.Name, item.ID)
	description += fmt.Sprintf("Description: %s\n\r", item.Description)
	description += fmt.Sprintf("Mass: %.2f\n\r", item.Mass)
	description += fmt.Sprintf("Value: %d\n\r", item.Value)
	description += fmt.Sprintf("Stackable: %v\n\r", item.Stackable)
	if item.Stackable {
		description += fmt.Sprintf("Quantity: %d/%d\n\r", item.Quantity, item.MaxStack)
	}

	if item.Wearable {
		description += fmt.Sprintf("Wearable on: %s\n\r", strings.Join(item.WornOn, ", "))
		if item.IsWorn {
			description += "This item is currently being worn.\n\r"
		}
	}

	if item.Container {
		description += "This is a container.\n\r"
		if len(item.Contents) > 0 {
			description += "It contains:\n\r"
			for _, contentItem := range item.Contents {
				description += fmt.Sprintf("  - %s (ID: %s)\n\r", contentItem.Name, contentItem.ID)
			}
		} else {
			description += "It is empty.\n\r"
		}
	}

	if len(item.Verbs) > 0 {
		description += "Special actions:\n\r"
		for verb, action := range item.Verbs {
			description += fmt.Sprintf("  %s: %s\n\r", verb, action)
		}
	}

	if len(item.TraitMods) > 0 {
		description += "Trait Modifications:\n\r"
		for trait, mod := range item.TraitMods {
			description += fmt.Sprintf("  %s: %d\n\r", trait, mod)
		}
	}

	if len(item.Metadata) > 0 {
		description += "Additional Information:\n\r"
		for key, value := range item.Metadata {
			description += fmt.Sprintf("  %s: %s\n\r", key, value)
		}
	}

	character.Player.ToPlayer <- description
	return false
}

func ExecuteHelpCommand(character *Character, tokens []string) bool {

	log.Printf("Player %s is requesting help", character.Player.Name)

	helpMessage := "\n\rAvailable Commands:" +
		"\n\rhelp - Display available commands" +
		"\n\rshow - Display character information" +
		"\n\rsay <message> - Say something to all players" +
		"\n\rlook - Look around the room" +
		"\n\rgo <direction> - Move in a direction" +
		"\n\rtake <item> - Take an item from the room" +
		"\n\rdrop <item> - Drop a held item" +
		"\n\rwear <item> - Wear an item from your inventory" +
		"\n\rremove <item> - Remove a worn item" +
		"\n\rexamine <item> - Get detailed information about an item" +
		"\n\rinventory (or i) - Check your inventory" +
		"\n\rwho - List all characters online" +
		"\n\rpassword <oldPassword> <newPassword> - Change your password" +
		"\n\rquit - Quit the game\n\r"

	character.Player.ToPlayer <- helpMessage
	return false
}