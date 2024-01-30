package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

var valid_commands []string = []string{"look", "go", "get", "drop", "inventory", "help", "quit", "say"}

// Function to check if a slice contains a specific string
func contains(slice []string, str string) bool {
	lowerStr := strings.ToLower(str)
	for _, v := range slice {
		if strings.ToLower(v) == lowerStr {
			return true
		}
	}
	return false
}

// Function to process the command
func validateCommand(command string, validCommands []string) (string, []string, error) {
	trimmedCommand := strings.TrimSpace(command)
	tokens := strings.Fields(trimmedCommand)

	if len(tokens) == 0 {
		return "", nil, errors.New("\n\rNo command entered.\n\r")
	}

	verb := ""

	// Convert all tokens to lowercase
	for i, token := range tokens {
		tokens[i] = strings.ToLower(token)
	}

	// Iterate through the tokens to find the first valid verb
	for _, token := range tokens {
		if contains(validCommands, token) {
			verb = token
			break
		}
	}

	if verb == "" {
		return verb, tokens, errors.New("\n\rI don't understand your command.")
	}

	return verb, tokens, nil
}

func executeCommand(player *Player, verb string, tokens []string) bool {

	command := strings.ToLower(verb)

	switch command {
	case "quit":
		return executeQuitCommand(player)

	case "say":
		return executeSayCommand(player, tokens)

	case "help":
		return executeHelpCommand(player)

	default:
		player.ToPlayer <- "\n\rCommand not yet implemented.\n\r"
	}

	return false // Indicate that the loop should continue
}

func executeQuitCommand(player *Player) bool {
	log.Printf("Player %s is quitting", player.Name)
	player.ToPlayer <- "\n\rGoodbye!"
	return true // Indicate that the loop should be exited
}

func executeSayCommand(player *Player, tokens []string) bool {
	if len(tokens) < 2 {
		player.ToPlayer <- "\n\rWhat do you want to say?\n\r"
		return false
	}

	message := strings.Join(tokens[1:], " ")
	broadcastMessage := fmt.Sprintf("\n\r%s says: %s\n\r", player.Name, message)

	player.Server.Mutex.Lock()
	for _, p := range player.Server.Players {
		if p != player {
			// Send message and prompt to other players
			p.ToPlayer <- broadcastMessage + p.Prompt
		}
	}
	player.Server.Mutex.Unlock()

	// Send only the broadcast message to the player who issued the command
	player.ToPlayer <- fmt.Sprintf("\n\rYou say: %s\n\r", message)

	return false
}

func executeHelpCommand(player *Player) bool {
	helpMessage := "\n\rAvailable Commands:" +
		"\n\rquit - Quit the game" +
		"\n\rsay <message> - Say something to all players" +
		"\n\rhelp - Display available commands\n\r"

	player.ToPlayer <- helpMessage
	return false
}
