package main

import (
	"errors"
	"fmt"
	"log"
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
	// Additional commands can be added here as needed.
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
		tokens[i] = strings.ToLower(token)
	}
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

func executeHelpCommand(character *Character, tokens []string) bool {
	helpMessage := "\n\rAvailable Commands:" +
		"\n\rquit - Quit the game" +
		"\n\rsay <message> - Say something to all players" +
		"\n\rlook - Look around the room" +
		"\n\rgo <direction> - Move in a direction" +
		"\n\rhelp - Display available commands\n\r"

	character.Player.ToPlayer <- helpMessage
	return false
}
