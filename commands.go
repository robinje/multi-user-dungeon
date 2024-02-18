package main

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
)

type CommandHandler func(character *Character, tokens []string) bool

var commandHandlers = map[string]CommandHandler{
	"quit":      executeQuitCommand,
	"look":      executeLookCommand,
	"say":       executeSayCommand,
	"go":        executeGoCommand,
	"help":      executeHelpCommand,
	"who":       executeWhoCommand,
	"password":  executePasswordCommand,
	"challenge": executeChallengeCommand,
	"\"":        executeSayCommand,  // Allow for double quotes to be used as a shortcut for the say command
	"'":         executeSayCommand,  // Allow for single quotes to be used as a shortcut for the say command
	"q!":        executeQuitCommand, // Allow for q! to be used as a shortcut for the quit command
	"fuck":      executeQuitCommand,
}

func validateCommand(command string, commandHandlers map[string]CommandHandler) (string, []string, error) {
	trimmedCommand := strings.TrimSpace(command)
	tokens := strings.Fields(trimmedCommand)

	if len(tokens) == 0 {
		return "", nil, errors.New("\n\rNo command entered.\n\r")
	}

	verb := strings.ToLower(tokens[0])
	if _, exists := commandHandlers[verb]; !exists {
		return "", tokens, fmt.Errorf("command not understood")
	}

	return verb, tokens, nil
}

func executeCommand(character *Character, verb string, tokens []string) bool {
	handler, ok := commandHandlers[verb]
	if !ok {
		character.Player.ToPlayer <- "\n\rCommand not yet implemented or recognized.\n\r"
		return false
	}
	return handler(character, tokens)
}

func executeQuitCommand(character *Character, tokens []string) bool {
	log.Printf("Player %s is quitting", character.Player.Name)
	character.Player.ToPlayer <- "\n\rGoodbye!"
	character.Room.SendRoomMessage(fmt.Sprintf("\n\r%s has left the game.\n\r", character.Name))

	return true // Indicate that the loop should be exited
}

func executeSayCommand(character *Character, tokens []string) bool {
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

func executeLookCommand(character *Character, tokens []string) bool {
	room := character.Room
	character.Player.ToPlayer <- room.RoomInfo(character)
	return false
}

func executeGoCommand(character *Character, tokens []string) bool {
	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rWhich direction do you want to go?\n\r"
		return false
	}

	direction := tokens[1]
	character.Move(direction)
	return false
}

func executeChallengeCommand(character *Character, tokens []string) bool {
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
	outcome := Challenge(attackerScore, defenderScore)

	// Provide feedback to the player based on the challenge outcome
	feedbackMessage := fmt.Sprintf("\n\rChallenge outcome: %f\n\r", outcome)
	character.Player.ToPlayer <- feedbackMessage

	return false
}

func executeWhoCommand(character *Character, tokens []string) bool {
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

func executePasswordCommand(character *Character, tokens []string) bool {
	if len(tokens) != 3 {
		character.Player.ToPlayer <- "\n\rUsage: password <oldPassword> <newPassword>\n\r"
		return false
	}

	oldPassword := tokens[1]
	newPassword := tokens[2]

	// Call the Server's method to change the password
	success, err := character.Server.ChangeUserPassword(character.Player.Name, oldPassword, newPassword)
	if err != nil {
		log.Printf("Failed to change password for user %s: %v", character.Player.Name, err)
		character.Player.ToPlayer <- "\n\rFailed to change password. Please try again.\n\r"
		return false
	}

	if success {
		character.Player.ToPlayer <- "\n\rPassword changed successfully.\n\r"
	} else {
		// This path might be redundant, as an error should already indicate failure
		character.Player.ToPlayer <- "\n\rFailed to change password for an unknown reason.\n\r"
	}

	return false // Keep the command loop running
}

func executeHelpCommand(character *Character, tokens []string) bool {
	helpMessage := "\n\rAvailable Commands:" +
		"\n\rhelp - Display available commands" +
		"\n\rsay <message> - Say something to all players" +
		"\n\rlook - Look around the room" +
		"\n\rgo <direction> - Move in a direction" +
		"\n\rwho - List all character online" +
		"\n\rpassword <oldPassword> <newPassword> - Change your password" +
		"\n\rquit - Quit the game\n\r"

	character.Player.ToPlayer <- helpMessage
	return false
}
