package main

import (
	"errors"
	"strings"
)

var valid_commands []string = []string{"look", "go", "get", "drop", "inventory", "quit"}

// Function to check if a slice contains a specific string
func contains(slice []string, str string) bool {
	// Convert the target string to lowercase
	lowerStr := strings.ToLower(str)
	for _, v := range slice {
		// Compare with the lowercase version of the slice strings
		if strings.ToLower(v) == lowerStr {
			return true
		}
	}
	return false
}

// Function to process the command
func validateCommand(command string, validCommands []string) ([]string, error) {
	// Trim the text of the command
	trimmedCommand := strings.TrimSpace(command)

	// Tokenize the command
	tokens := strings.Fields(trimmedCommand)

	// Check if there are any tokens to process
	if len(tokens) == 0 {
		return nil, errors.New("No command entered.")
	}

	// Initialize verb as an empty string
	verb := ""

	// Iterate through the tokens to find the first valid verb
	for _, token := range tokens {
		if contains(validCommands, token) {
			verb = token
			break
		}
	}

	// Check if a valid verb was found
	if verb == "" {
		return nil, errors.New("I don't understand your command.")
	}

	// Process the valid command here (this part can be customized as needed)
	return tokens, nil
}
