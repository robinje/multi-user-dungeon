package main

import (
	"errors"
	"log"
	"strings"
)

var valid_commands []string = []string{"look", "go", "get", "drop", "inventory", "quit", "say"}

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
		return "", nil, errors.New("No command entered.")
	}

	verb := ""

	// Iterate through the tokens to find the first valid verb
	for _, token := range tokens {
		if contains(validCommands, token) {
			verb = token
			break
		}
	}

	if verb == "" {
		return verb, tokens, errors.New("I don't understand your command.")
	}

	return verb, tokens, nil
}

func executeCommand(player *Player, verb string, tokens []string) bool {
	command := strings.ToLower(verb)

	switch command {
	case "quit":
		log.Printf("Player %s is quitting", player.Name)
		player.ToPlayer <- "Goodbye!\n\r"
		return true // Indicate that the loop should be exited

	default:
		player.ToPlayer <- "Command not yet implemented.\n\r"
	}
	return false // Indicate that the loop should continue
}
