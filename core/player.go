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
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

func (k *KeyPair) WritePlayer(player *Player) error {
	pd := PlayerData{
		Name:          player.Name,
		CharacterList: make(map[string]string),
	}

	// Convert UUID to string
	for charName, charID := range player.CharacterList {
		pd.CharacterList[charName] = charID.String()
	}

	av, err := dynamodbattribute.MarshalMap(pd)
	if err != nil {
		return fmt.Errorf("error marshalling player data: %w", err)
	}

	key := map[string]*dynamodb.AttributeValue{
		"Name": {S: aws.String(player.Name)},
	}

	err = k.Put("players", key, av)
	if err != nil {
		return fmt.Errorf("error storing player data: %w", err)
	}

	Logger.Info("Successfully wrote player data", "playerName", player.Name, "characterCount", len(player.CharacterList))
	return nil
}

func (k *KeyPair) ReadPlayer(playerName string) (string, map[string]uuid.UUID, error) {
	key := map[string]*dynamodb.AttributeValue{
		"Name": {S: aws.String(playerName)},
	}

	var pd PlayerData
	err := k.Get("players", key, &pd)
	if err != nil {
		Logger.Error("Error reading player data", "error", err)
		return "", nil, fmt.Errorf("player not found")
	}

	if pd.Name == "" {
		pd.Name = playerName
	}

	characterList := make(map[string]uuid.UUID)
	for name, idString := range pd.CharacterList {
		id, err := uuid.Parse(idString)
		if err != nil {
			Logger.Error("Error parsing UUID for character", "characterName", name, "error", err)
			continue
		}
		characterList[name] = id
	}

	Logger.Info("Successfully read player data", "playerName", pd.Name, "characterCount", len(characterList))
	return pd.Name, characterList, nil
}

func PlayerInput(p *Player) {
	Logger.Info("Player input goroutine started", "playerName", p.Name)

	var inputBuffer bytes.Buffer
	const maxBufferSize = 1024 // Maximum input size in bytes

	reader := bufio.NewReader(p.Connection)

	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				Logger.Info("Player disconnected", "playerName", p.Name, "error", err)
				p.PlayerError <- err
				break
			} else {
				Logger.Error("Error reading from player", "playerName", p.Name, "error", err)
				p.PlayerError <- err
				continue
			}
		}

		if p.Echo && char != '\n' && char != '\r' {
			if _, err := p.Connection.Write([]byte(string(char))); err != nil {
				Logger.Error("Failed to echo character to player", "playerName", p.Name, "error", err)
			}
		}

		if char == '\n' || char == '\r' {
			if inputBuffer.Len() > 0 {
				p.FromPlayer <- inputBuffer.String()
				inputBuffer.Reset()
			}
			continue
		}

		if inputBuffer.Len() >= maxBufferSize {
			Logger.Warn("Input buffer overflow, discarding input", "playerName", p.Name)
			p.ToPlayer <- "\n\rInput too long, discarded.\n\r"
			inputBuffer.Reset()
			continue
		}

		inputBuffer.WriteRune(char)
	}

	close(p.FromPlayer)
}

func PlayerOutput(p *Player) {

	Logger.Info("Player output goroutine started", "playerName", p.Name)

	for message := range p.ToPlayer {
		// Append carriage return and newline for SSH protocol compatibility
		messageToSend := message
		if _, err := p.Connection.Write([]byte(messageToSend)); err != nil {
			Logger.Error("Failed to send message to player", "playerName", p.Name, "error", err)
			// Consider whether to continue or break based on your error handling policy
			continue
		}
	}

	// Optionally, perform any cleanup here after the channel is closed and loop exits
	Logger.Info("Message channel closed for player", "playerName", p.Name)
}

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
					Logger.Info("Player issued command", "playerName", c.Player.Name, "command", strings.Join(tokens, " "))
				}
				lastCommand = ""
				if !shouldQuit {
					c.Player.ToPlayer <- c.Player.Prompt
				}
			}

		case inputLine, more := <-c.Player.FromPlayer:
			if !more {
				Logger.Info("Input channel closed for player", "playerName", c.Player.Name)
				shouldQuit = true
				break
			}
			lastCommand = strings.Replace(inputLine, "\n", "\n\r", -1)
		}
	}

	// Cleanup code
	close(c.Player.FromPlayer)

	c.Room.Mutex.Lock()
	delete(c.Room.Characters, c.ID)
	c.Room.Mutex.Unlock()

	c.Server.Mutex.Lock()
	delete(c.Server.Characters, c.ID)
	c.Server.Mutex.Unlock()

	err := c.Server.Database.WriteCharacter(c)
	if err != nil {
		Logger.Error("Error saving character", "characterName", c.Name, "error", err)
	}

	Logger.Info("Input loop ended for character", "characterName", c.Name)
}

func SelectCharacter(player *Player, server *Server) (*Character, error) {
	Logger.Info("Player is selecting a character", "playerName", player.Name)

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
		} else {
			player.ToPlayer <- "No existing characters found.\n\r"
		}
		player.ToPlayer <- "Enter the number of your choice: "
	}

	for {
		options = []string{} // Reset options for each iteration
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

		var character *Character
		if choice == 0 {
			character, err = CreateCharacter(player, server)
			if err != nil {
				player.ToPlayer <- fmt.Sprintf("Error creating character: %v\n\r", err)
				continue
			}
		} else if choice <= len(options) {
			characterName := options[choice-1]
			characterID := player.CharacterList[characterName]
			character, err = server.Database.LoadCharacter(characterID, player, server)
			if err != nil {
				Logger.Error("Error loading character for player", "characterName", characterName, "playerName", player.Name, "error", err)
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
			character.Room.Mutex.Lock()
			character.Room.Characters[character.ID] = character
			character.Room.Mutex.Unlock()

			// Notify the room that the character has entered
			SendRoomMessage(character.Room, fmt.Sprintf("\n\r%s has arrived.\n\r", character.Name))
		}

		Logger.Info("Character selected and added to server", "characterName", character.Name, "characterID", character.ID)

		return character, nil
	}
}

func CreateCharacter(player *Player, server *Server) (*Character, error) {
	Logger.Info("Player is creating a new character", "playerName", player.Name)

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

	if server.CharacterNameExists(charName) {
		return nil, fmt.Errorf("character name already exists")
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

	Logger.Info("Creating character", "characterName", charName)

	room, ok := server.Rooms[1]
	if !ok {
		room, ok = server.Rooms[0]
		if !ok {
			return nil, fmt.Errorf("no starting room found")
		}
	}

	character := server.NewCharacter(charName, player, room, selectedArchetype)

	player.Mutex.Lock()
	if player.CharacterList == nil {
		player.CharacterList = make(map[string]uuid.UUID)
	}
	player.CharacterList[charName] = character.ID
	player.Mutex.Unlock()

	Logger.Info("Added character to player's character list", "characterName", charName, "characterID", character.ID, "playerName", player.Name)

	// Save the character to the database
	err := server.Database.WriteCharacter(character)
	if err != nil {
		Logger.Error("Error saving character to database", "characterName", charName, "error", err)
		return nil, fmt.Errorf("failed to save character to database")
	}

	// Save the updated player data
	err = server.Database.WritePlayer(player)
	if err != nil {
		Logger.Error("Error saving player data", "playerName", player.Name, "error", err)
		return nil, fmt.Errorf("failed to save player data")
	}

	Logger.Info("Successfully created and saved character for player", "characterName", charName, "characterID", character.ID, "playerName", player.Name)

	return character, nil
}
