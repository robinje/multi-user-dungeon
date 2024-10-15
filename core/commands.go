package core

import (
	"errors"
	"fmt"
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
	"take":      ExecuteTakeCommand,
	"get":       ExecuteTakeCommand, // Alias for take command
	"drop":      ExecuteDropCommand,
	"inventory": ExecuteInventoryCommand,
	"wear":      ExecuteWearCommand,
	"remove":    ExecuteRemoveCommand,
	"examine":   ExecuteExamineCommand,
	"assess":    ExecuteAssessCommand,
	"i":         ExecuteInventoryCommand, // Alias for inventory command
	"inv":       ExecuteInventoryCommand, // Alias for inventory command
	"\"":        ExecuteSayCommand,       // Allow for double quotes to be used as a shortcut for the say command
	"'":         ExecuteSayCommand,       // Allow for single quotes to be used as a shortcut for the say command
	"q!":        ExecuteQuitCommand,      // Allow for q! to be used as a shortcut for the quit command
}

func ValidateCommand(command string) (string, []string, error) {

	Logger.Debug("Received command", "command", command)

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

	Logger.Debug("Executing command", "verb", verb)

	handler, ok := CommandHandlers[verb]
	if !ok {
		character.Player.ToPlayer <- "\n\rCommand not yet implemented or recognized.\n\r"
		return false
	}
	return handler(character, tokens)
}

func ExecuteQuitCommand(character *Character, tokens []string) bool {
	Logger.Info("Player is quitting", "playerName", character.Player.PlayerID)

	// Send goodbye message
	character.Player.ToPlayer <- "\n\rGoodbye!"

	// Remove character from the room
	character.Room.Mutex.Lock()
	delete(character.Room.Characters, character.ID)
	character.Room.Mutex.Unlock()

	// Remove character from the server's active characters
	character.Server.Mutex.Lock()
	delete(character.Server.Characters, character.ID)
	character.Server.Mutex.Unlock()

	// Notify room
	SendRoomMessage(character.Room, fmt.Sprintf("\n\r%s has left.\n\r", character.Name))

	// Save character state to database
	character.Mutex.Lock()
	err := character.Server.Database.WriteCharacter(character)
	if err != nil {
		Logger.Error("Error saving character state on quit", "characterName", character.Name, "error", err)
	}
	character.Mutex.Unlock()

	Logger.Info("Player has successfully quit", "playerName", character.Player.PlayerID)

	return true // Indicate that the loop should be exited
}

func ExecuteSayCommand(character *Character, tokens []string) bool {

	Logger.Info("Player is saying something", "playerName", character.Player.PlayerID)

	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rWhat do you want to say?\n\r"
		return false
	}

	message := strings.Join(tokens[1:], " ")
	broadcastMessage := fmt.Sprintf("\n\r%s says %s\n\r", character.Name, message)

	for _, c := range character.Room.Characters {
		if c != character {
			// Send message to other characters in the room
			c.Player.ToPlayer <- broadcastMessage
			c.Player.ToPlayer <- c.Player.Prompt
		}
	}

	// Send only the broadcast message to the player who issued the command
	character.Player.ToPlayer <- fmt.Sprintf("\n\rYou say %s\n\r", message)

	return false
}

func ExecuteLookCommand(character *Character, tokens []string) bool {

	Logger.Info("Player is looking around", "playerName", character.Player.PlayerID)

	room := character.Room
	character.Player.ToPlayer <- RoomInfo(room, character)
	return false
}

func ExecuteGoCommand(character *Character, tokens []string) bool {

	Logger.Info("Player is attempting to move", "playerName", character.Player.PlayerID)

	if !character.CanEscape() {
		character.Player.ToPlayer <- "\n\rYou can't escape!\n\r"
		return false
	}

	// Ensure the correct number of arguments are provided

	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rWhich direction do you want to go?\n\r"
		return false
	}

	direction := tokens[1]
	Move(character, direction)

	character.ExitCombat()

	return false
}

func ExecuteChallengeCommand(character *Character, tokens []string) bool {

	Logger.Info("Player is attempting a challenge", "playerName", character.Player.PlayerID)

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
	Logger.Info("Player is listing all characters online", "playerName", character.Player.PlayerID)

	// Retrieve the server instance from the character
	server := character.Server

	characterNames := make([]string, 0, len(server.Characters))
	for _, char := range server.Characters {
		characterNames = append(characterNames, char.Name)
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

	Logger.Info("Player is attempting to change their password", "playerName", character.Player.PlayerID)

	if len(tokens) != 3 {
		character.Player.ToPlayer <- "\n\rUsage: password <oldPassword> <newPassword>\n\r"
		return false
	}

	oldPassword := tokens[1]
	newPassword := tokens[2]

	err := ChangePassword(character.Server, character.Player.PlayerID, oldPassword, newPassword)
	if err != nil {
		Logger.Error("Failed to change password for user", "playerName", character.Player.PlayerID, "error", err)
		character.Player.ToPlayer <- "\n\rFailed to change password. Please try again.\n\r"
		return false
	}

	character.Player.ToPlayer <- "\n\rPassword changed successfully.\n\r"
	return false // Keep the command loop running
}

func ExecuteShowCommand(character *Character, tokens []string) bool {

	Logger.Info("Player is displaying character information", "playerName", character.Player.PlayerID)

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

	// Try to place the item in the right hand first, then the left hand if right is occupied
	var handSlot string
	if character.Inventory["right_hand"] == nil {
		handSlot = "right_hand"
	} else if character.Inventory["left_hand"] == nil {
		handSlot = "left_hand"
	}

	if handSlot == "" {
		character.Player.ToPlayer <- "\n\rYour hands are full. You need a free hand to pick up an item.\n\r"
		return false
	}

	character.Room.RemoveItem(itemToTake)
	character.Mutex.Lock()
	character.Inventory[handSlot] = itemToTake
	character.Mutex.Unlock()

	SendRoomMessage(character.Room, fmt.Sprintf("\n\r%s picks up %s.\n\r", character.Name, itemToTake.Name))
	character.Player.ToPlayer <- fmt.Sprintf("\n\rYou take %s and hold it in your %s.\n\r", itemToTake.Name, strings.Replace(handSlot, "_", " ", -1))
	return false
}

func ExecuteInventoryCommand(character *Character, tokens []string) bool {

	Logger.Info("Player is checking their inventory", "playerName", character.Player.PlayerID)

	inventoryList := ListInventory(character)
	character.Player.ToPlayer <- inventoryList
	return false
}

func ExecuteDropCommand(character *Character, tokens []string) bool {
	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rUsage: drop <item name>\n\r"
		return false
	}

	itemName := strings.ToLower(strings.Join(tokens[1:], " "))
	var itemToDrop *Item
	var handSlot string

	// Check if the item is in a hand slot
	for slot, item := range character.Inventory {
		if (slot == "left_hand" || slot == "right_hand") && strings.Contains(strings.ToLower(item.Name), itemName) {
			itemToDrop = item
			handSlot = slot
			break
		}
	}

	if itemToDrop == nil {
		character.Player.ToPlayer <- "\n\rYou're not holding that item.\n\r"
		return false
	}
	character.Mutex.Lock()
	delete(character.Inventory, handSlot)
	character.Mutex.Unlock()
	character.Room.Mutex.Lock()
	character.Room.AddItem(itemToDrop)
	character.Room.Mutex.Unlock()

	character.Player.ToPlayer <- fmt.Sprintf("\n\rYou drop %s.\n\r", itemToDrop.Name)
	SendRoomMessage(character.Room, fmt.Sprintf("\n\r%s drops %s.\n\r", character.Name, itemToDrop.Name))
	return false
}

func ExecuteWearCommand(character *Character, tokens []string) bool {

	Logger.Info("Player is attempting to wear an item", "playerName", character.Player.PlayerID)

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
	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rUsage: remove <item name>\n\r"
		return false
	}

	itemName := strings.ToLower(strings.Join(tokens[1:], " "))
	var itemToRemove *Item

	for _, item := range character.Inventory {
		if item != nil && item.IsWorn && strings.Contains(strings.ToLower(item.Name), itemName) {
			itemToRemove = item
			break
		}
	}

	if itemToRemove == nil {
		character.Player.ToPlayer <- "\n\rYou're not wearing that item.\n\r"
		return false
	}

	err := RemoveWornItem(character, itemToRemove)
	if err != nil {
		character.Player.ToPlayer <- fmt.Sprintf("\n\r%s\n\r", err.Error())
		return false
	}

	character.Player.ToPlayer <- fmt.Sprintf("\n\rYou remove %s.\n\r", itemToRemove.Name)
	SendRoomMessage(character.Room, fmt.Sprintf("\n\r%s removes %s.\n\r", character.Name, itemToRemove.Name))
	return false
}

func ExecuteExamineCommand(character *Character, tokens []string) bool {

	Logger.Info("Player is examining an item", "playerName", character.Player.PlayerID)

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

func ExecuteAssessCommand(character *Character, tokens []string) bool {
	Logger.Info("Player is assessing combat situation", "playerName", character.Player.PlayerID)

	if !character.IsInCombat() {
		character.Player.ToPlayer <- "\n\rYou are not currently in combat.\n\r"
		return false
	}

	var assessment strings.Builder
	assessment.WriteString("\n\rCombat Assessment:\n\r")

	if len(character.CombatRange) == 0 {
		assessment.WriteString("You are in combat, but not engaged with any specific opponents.\n\r")
	} else {
		for targetID, distance := range character.CombatRange {
			targetCharacter := character.Server.Characters[targetID]
			if targetCharacter == nil {
				continue // Skip if the character is not found (should not happen in normal circumstances)
			}

			var rangeDescription string
			switch distance {
			case 0:
				rangeDescription = "far"
			case 1:
				rangeDescription = "pole"
			case 2:
				rangeDescription = "melee"
			default:
				rangeDescription = "unknown"
			}

			facingInfo := ""
			if targetCharacter.GetFacing() == character {
				facingInfo = " and is facing you"
			}

			assessment.WriteString(fmt.Sprintf("%s is at %s range%s.\n\r", targetCharacter.Name, rangeDescription, facingInfo))
		}
	}

	if character.CanEscape() {
		assessment.WriteString("You can attempt to escape from combat.\n\r")
	} else {
		assessment.WriteString("You cannot escape from combat at this time.\n\r")
	}

	character.Player.ToPlayer <- assessment.String()
	return false
}

func ExecuteFaceCommand(character *Character, tokens []string) bool {
	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rUsage: face <character name>\n\r"
		return false
	}

	targetName := strings.Join(tokens[1:], " ")
	var targetCharacter *Character

	// Find the target character in the same room
	for _, c := range character.Room.Characters {
		if strings.EqualFold(c.Name, targetName) {
			targetCharacter = c
			break
		}
	}

	if targetCharacter == nil {
		character.Player.ToPlayer <- fmt.Sprintf("\n\rYou don't see %s here.\n\r", targetName)
		return false
	}

	// Set facing for the character executing the command
	character.SetFacing(targetCharacter)

	// Enter combat and set range to far (0) for both characters
	character.EnterCombat()
	targetCharacter.EnterCombat()

	character.SetCombatRange(targetCharacter, 0) // 0 represents far range
	targetCharacter.SetCombatRange(character, 0) // Reciprocal setting

	character.Player.ToPlayer <- fmt.Sprintf("\n\rYou are now facing %s at far range.\n\r", targetCharacter.Name)

	// Notify the target character
	targetCharacter.Player.ToPlayer <- fmt.Sprintf("\n\r%s is now facing you at far range.\n\r", character.Name)
	targetCharacter.Player.ToPlayer <- targetCharacter.Player.Prompt

	return false
}

func ExecuteHelpCommand(character *Character, tokens []string) bool {

	Logger.Info("Player is requesting help", "playerName", character.Player.PlayerID)

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
		"\n\rassess - Assess your current combat situation" +
		"\n\rface <character> - Face a character in the room" +
		"\n\rwho - List all characters online" +
		"\n\rpassword <oldPassword> <newPassword> - Change your password" +
		"\n\rquit - Quit the game\n\r"

	character.Player.ToPlayer <- helpMessage
	return false
}
