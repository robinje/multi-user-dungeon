package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
)

// WritePlayer stores the player data into the DynamoDB database.
func (k *KeyPair) WritePlayer(player *Player) error {
	pd := PlayerData{
		PlayerID:      player.PlayerID,
		CharacterList: make(map[string]string),
		SeenMotDs:     make([]string, len(player.SeenMotDs)),
	}

	// Convert UUIDs to strings for CharacterList
	for charName, charID := range player.CharacterList {
		pd.CharacterList[charName] = charID.String()
	}

	// Convert UUIDs to strings for SeenMotDs
	for i, motdID := range player.SeenMotDs {
		pd.SeenMotDs[i] = motdID.String()
	}

	// Write the player data to the DynamoDB table with proper error handling
	err := k.Put("players", pd)
	if err != nil {
		Logger.Error("Error storing player data", "playerName", player.PlayerID, "error", err)
		return fmt.Errorf("error storing player data: %w", err)
	}

	Logger.Info("Successfully wrote player data", "playerName", player.PlayerID, "characterCount", len(player.CharacterList), "seenMotDCount", len(player.SeenMotDs))
	return nil
}

// ReadPlayer retrieves the player data from the DynamoDB database.
func (k *KeyPair) ReadPlayer(playerName string) (string, map[string]uuid.UUID, []uuid.UUID, error) {
	key := map[string]*dynamodb.AttributeValue{
		"PlayerID": {S: aws.String(playerName)},
	}

	var pd PlayerData

	// Read the player data from the DynamoDB table with proper error handling
	err := k.Get("players", key, &pd)
	if err != nil {
		Logger.Error("Error reading player data", "playerName", playerName, "error", err)
		return "", nil, nil, fmt.Errorf("player not found")
	}

	// Convert character IDs from strings to UUIDs
	characterList := make(map[string]uuid.UUID)
	for name, idString := range pd.CharacterList {
		id, err := uuid.Parse(idString)
		if err != nil {
			Logger.Error("Error parsing UUID for character", "characterName", name, "error", err)
			continue // Skip invalid UUIDs
		}
		characterList[name] = id
	}

	// Convert SeenMotDs from strings to UUIDs
	seenMotDs := make([]uuid.UUID, 0, len(pd.SeenMotDs))
	for _, idString := range pd.SeenMotDs {
		id, err := uuid.Parse(idString)
		if err != nil {
			Logger.Error("Error parsing UUID for seen MOTD", "idString", idString, "error", err)
			continue // Skip invalid UUIDs
		}
		seenMotDs = append(seenMotDs, id)
	}

	Logger.Info("Successfully read player data", "playerName", pd.PlayerID, "characterCount", len(characterList), "seenMotDCount", len(seenMotDs))
	return pd.PlayerID, characterList, seenMotDs, nil
}

// PlayerInput handles the player's input in a separate goroutine.
// It reads input from the player's SSH connection and sends it to the FromPlayer channel.
func PlayerInput(p *Player) {
	Logger.Info("Player input goroutine started", "playerName", p.PlayerID)

	var inputBuffer bytes.Buffer
	const maxBufferSize = 1024 // Maximum input size in bytes

	reader := bufio.NewReader(p.Connection)

	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				Logger.Info("Player disconnected", "playerName", p.PlayerID)
				p.PlayerError <- err
				break
			} else {
				Logger.Error("Error reading from player", "playerName", p.PlayerID, "error", err)
				p.PlayerError <- err
				continue
			}
		}

		if p.Echo && char != '\n' && char != '\r' {
			if _, err := p.Connection.Write([]byte(string(char))); err != nil {
				Logger.Error("Failed to echo character to player", "playerName", p.PlayerID, "error", err)
			}
		}

		if char == '\n' || char == '\r' {
			if inputBuffer.Len() > 0 {
				// Send the accumulated input to the FromPlayer channel
				p.FromPlayer <- inputBuffer.String()
				inputBuffer.Reset()
			}
			continue
		}

		if inputBuffer.Len() >= maxBufferSize {
			Logger.Warn("Input buffer overflow, discarding input", "playerName", p.PlayerID)
			p.ToPlayer <- "\n\rInput too long, discarded.\n\r"
			inputBuffer.Reset()
			continue
		}

		inputBuffer.WriteRune(char)
	}

	// Close the FromPlayer channel to signal that no more input will be sent
	close(p.FromPlayer)
}

// PlayerOutput handles sending messages to the player in a separate goroutine.
// It reads messages from the ToPlayer channel and writes them to the player's SSH connection.
func PlayerOutput(p *Player) {
	Logger.Info("Player output goroutine started", "playerName", p.PlayerID)

	for message := range p.ToPlayer {
		// Write the message to the player's SSH connection
		if _, err := p.Connection.Write([]byte(message)); err != nil {
			Logger.Error("Failed to send message to player", "playerName", p.PlayerID, "error", err)
			// Decide whether to continue or break based on your error handling policy
			continue
		}
	}

	// Optional cleanup after the channel is closed
	Logger.Info("Message channel closed for player", "playerName", p.PlayerID)
}

// InputLoop is the main loop that handles player commands.
// It reads commands from the player's input and executes them accordingly.
func InputLoop(c *Character) {
	Logger.Info("Starting input loop for character", "characterName", c.Name)

	// Initially execute the look command with no additional tokens
	ExecuteLookCommand(c, []string{})

	// Send initial prompt to player
	c.Player.ToPlayer <- c.Player.Prompt

	// Create a ticker that ticks once per second
	commandTicker := time.NewTicker(time.Second)
	defer commandTicker.Stop()

	var lastCommand string
	shouldQuit := false

	for !shouldQuit {
		select {
		case <-commandTicker.C:
			if lastCommand != "" {
				verb, tokens, err := ValidateCommand(strings.TrimSpace(lastCommand))
				if err != nil {
					c.Player.ToPlayer <- err.Error() + "\n\r"
				} else {
					// Execute the command
					shouldQuit = ExecuteCommand(c, verb, tokens)
					Logger.Info("Player issued command", "playerName", c.Player.PlayerID, "command", strings.Join(tokens, " "))
				}
				lastCommand = ""
				if !shouldQuit {
					c.Player.ToPlayer <- c.Player.Prompt
				}
			}

		case inputLine, more := <-c.Player.FromPlayer:
			if !more {
				Logger.Info("Input channel closed for player", "playerName", c.Player.PlayerID)
				shouldQuit = true
				break
			}
			lastCommand = strings.Replace(inputLine, "\n", "\n\r", -1)
		}
	}

	// Cleanup code
	close(c.Player.FromPlayer)

	// Remove character from room and server
	c.Room.Mutex.Lock()
	delete(c.Room.Characters, c.ID)
	c.Room.Mutex.Unlock()

	c.Server.Mutex.Lock()
	delete(c.Server.Characters, c.ID)
	c.Server.Mutex.Unlock()

	// Save character state to the database
	err := c.Server.Database.WriteCharacter(c)
	if err != nil {
		Logger.Error("Error saving character", "characterName", c.Name, "error", err)
	}

	Logger.Info("Input loop ended for character", "characterName", c.Name)
}

// SelectCharacter handles the character selection process for a player.
// It presents the player with options to select or create a character.
func SelectCharacter(player *Player, server *Server) (*Character, error) {
	Logger.Info("Player is selecting a character", "playerName", player.PlayerID)

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
			player.ToPlayer <- "X: Delete a character\n\r"
		} else {
			player.ToPlayer <- "No existing characters found.\n\r"
		}
		player.ToPlayer <- "Enter the number of your choice or 'X' to delete: "
	}

	for {
		options = []string{} // Reset options for each iteration
		sendCharacterOptions()

		input, ok := <-player.FromPlayer
		if !ok {
			Logger.Error("Failed to receive input from player", "playerName", player.PlayerID)
			return nil, fmt.Errorf("failed to receive input")
		}

		input = strings.TrimSpace(strings.ToUpper(input))

		if input == "X" && len(player.CharacterList) > 0 {
			// Handle character deletion
			player.ToPlayer <- "Select a character to delete:\n\r"
			for i, name := range options {
				player.ToPlayer <- fmt.Sprintf("%d: %s\n\r", i+1, name)
			}
			player.ToPlayer <- "Enter the number of the character to delete: "

			deleteChoice, ok := <-player.FromPlayer
			if !ok {
				Logger.Error("Failed to receive delete choice from player", "playerName", player.PlayerID)
				return nil, fmt.Errorf("failed to receive delete choice")
			}

			deleteIndex, err := strconv.Atoi(strings.TrimSpace(deleteChoice))
			if err != nil || deleteIndex < 1 || deleteIndex > len(options) {
				player.ToPlayer <- "Invalid choice. Returning to character selection.\n\r"
				continue
			}

			characterToDelete := options[deleteIndex-1]
			err = server.DeleteCharacter(player, characterToDelete)
			if err != nil {
				Logger.Error("Failed to delete character", "characterName", characterToDelete, "error", err)
				player.ToPlayer <- fmt.Sprintf("Failed to delete character: %v\n\r", err)
			} else {
				player.ToPlayer <- fmt.Sprintf("Character '%s' has been deleted.\n\r", characterToDelete)
			}
			continue
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 0 || choice > len(options) {
			player.ToPlayer <- "Invalid choice. Please select a valid option.\n\r"
			continue
		}

		var character *Character
		if choice == 0 {
			character, err = server.CreateCharacter(player)
			if err != nil {
				player.ToPlayer <- fmt.Sprintf("\n\rError creating character: %v\n\r", err)
				continue
			}
		} else if choice <= len(options) {
			characterName := options[choice-1]
			characterID := player.CharacterList[characterName]
			character, err = server.Database.LoadCharacter(characterID, player, server)
			if err != nil {
				Logger.Error("Error loading character for player", "characterName", characterName, "playerName", player.PlayerID, "error", err)
				player.ToPlayer <- fmt.Sprintf("Error loading character: %v\n\r", err)
				continue
			}
		}

		if character == nil {
			player.ToPlayer <- "Failed to create or load character. Please try again.\n\r"
			continue
		}

		// Ensure the character is added to the server's character list
		server.Mutex.Lock()
		server.Characters[character.ID] = character
		server.Mutex.Unlock()

		// Add character to the room and notify other players
		if character.Room != nil {
			// Notify the room that the character has entered
			SendRoomMessage(character.Room, fmt.Sprintf("\n\r%s has arrived.\n\r", character.Name))

			character.Room.Mutex.Lock()
			character.Room.Characters[character.ID] = character
			character.Room.Mutex.Unlock()
		}

		Logger.Info("Character selected and added to server", "characterName", character.Name, "characterID", character.ID)

		return character, nil
	}
}

// CreateCharacter handles the character creation process for a player.
// It prompts the player for a character name and archetype, and initializes the character.
func (s *Server) CreateCharacter(player *Player) (*Character, error) {
	Logger.Info("Player is creating a new character", "playerName", player.PlayerID)

	player.ToPlayer <- "\n\rEnter your character name: "

	charName, ok := <-player.FromPlayer
	if !ok {
		Logger.Error("Failed to receive character name input", "playerName", player.PlayerID)
		return nil, fmt.Errorf("failed to receive character name input")
	}

	charName = strings.TrimSpace(charName)

	// Validate character name
	if len(charName) == 0 {
		player.ToPlayer <- "Character name cannot be empty.\n\r"
		return nil, fmt.Errorf("character name cannot be empty")
	}

	if len(charName) > 15 {
		player.ToPlayer <- "Character name must be 15 characters or fewer.\n\r"
		return nil, fmt.Errorf("character name must be 15 characters or fewer")
	}

	if s.CharacterNameExists(charName) {
		player.ToPlayer <- "Character name already exists. Please choose another name.\n\r"
		return nil, fmt.Errorf("character name already exists")
	}

	var selectedArchetype string

	// If archetypes are available, prompt the player to select one
	if s.Archetypes != nil && len(s.Archetypes.Archetypes) > 0 {
		for {
			selectionMsg := "\n\rSelect a character archetype.\n\r"
			archetypeOptions := make([]string, 0, len(s.Archetypes.Archetypes))
			for name, archetype := range s.Archetypes.Archetypes {
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
				Logger.Error("Failed to receive archetype selection", "playerName", player.PlayerID)
				return nil, fmt.Errorf("failed to receive archetype selection")
			}

			selectionNum, err := strconv.Atoi(strings.TrimSpace(selection))
			if err == nil && selectionNum >= 1 && selectionNum <= len(archetypeOptions) {
				selectedOption := archetypeOptions[selectionNum-1]
				selectedArchetype = strings.Split(selectedOption, " - ")[0]
				break
			} else {
				player.ToPlayer <- "Invalid selection. Please select a valid archetype number.\n\r"
			}
		}
	}

	Logger.Info("Creating character", "characterName", charName)

	// Attempt to find the starting room
	room, ok := s.Rooms[1]
	if !ok {
		Logger.Warn("Starting room not found, using default room", "startingRoomID", 1)

		// Attempt to find default room (room ID 0)
		room, ok = s.Rooms[0]
		if !ok {
			Logger.Error("No default room found", "defaultRoomID", 0)
			Logger.Info("Available rooms", "rooms", s.Rooms)
			player.ToPlayer <- "No starting or default room found. Please contact the administrator.\n\r"
			return nil, fmt.Errorf("no starting or default room found")
		}
	}

	// Create the new character
	character := s.NewCharacter(charName, player, room, selectedArchetype)

	player.Mutex.Lock()
	if player.CharacterList == nil {
		player.CharacterList = make(map[string]uuid.UUID)
	}
	player.CharacterList[charName] = character.ID
	player.Mutex.Unlock()

	Logger.Info("Added character to player's character list", "characterName", charName, "characterID", character.ID, "playerName", player.PlayerID)

	// Save the character to the database with error handling
	err := s.Database.WriteCharacter(character)
	if err != nil {
		Logger.Error("Error saving character to database", "characterName", charName, "error", err)
		player.ToPlayer <- "Error saving character to database. Please try again later.\n\r"
		return nil, fmt.Errorf("failed to save character to database: %w", err)
	}

	// Save the updated player data with error handling
	err = s.Database.WritePlayer(player)
	if err != nil {
		Logger.Error("Error saving player data", "playerName", player.PlayerID, "error", err)
		player.ToPlayer <- "Error saving player data. Please try again later.\n\r"
		return nil, fmt.Errorf("failed to save player data: %w", err)
	}

	Logger.Info("Successfully created and saved character for player", "characterName", charName, "characterID", character.ID, "playerName", player.PlayerID)

	return character, nil
}
