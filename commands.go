package main

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
)

var validCommands = []string{"look", "go", "get", "drop", "inventory", "help", "quit", "say"}

type CommandHandler func(character *Character, tokens []string) bool

var commandHandlers = map[string]CommandHandler{
	"quit": executeQuitCommand,
	"look": executeLookCommand,
	"say":  executeSayCommand,
	"go":   executeGoCommand,
	"help": executeHelpCommand,
	"who":  executeWhoCommand,
}

func contains(slice []string, str string) bool {
	lowerStr := strings.ToLower(str)
	for _, v := range slice {
		if strings.ToLower(v) == lowerStr {
			return true
		}
	}
	return false
}

func validateCommand(command string, validCommands []string) (string, []string, error) {
	trimmedCommand := strings.TrimSpace(command)
	tokens := strings.Fields(trimmedCommand)

	if len(tokens) == 0 {
		return "", nil, errors.New("\n\rNo command entered.\n\r")
	}

	verb := ""
	for i, token := range tokens {
		tokens[i] = token
	}
	for _, token := range tokens {
		if contains(validCommands, strings.ToLower(token)) {
			verb = strings.ToLower(token)
			break
		}
	}
	if verb == "" {
		return verb, tokens, errors.New("\n\rI don't understand your command.")
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

func executeHelpCommand(character *Character, tokens []string) bool {
	helpMessage := "\n\rAvailable Commands:" +
		"\n\rhelp - Display available commands" +
		"\n\rsay <message> - Say something to all players" +
		"\n\rlook - Look around the room" +
		"\n\rgo <direction> - Move in a direction" +
		"\n\rwho - List all character online" +
		"\n\rquit - Quit the game\n\r"

	character.Player.ToPlayer <- helpMessage
	return false
}
