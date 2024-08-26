package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
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

	log.Printf("Successfully wrote player data for %s with %d characters", player.Name, len(player.CharacterList))
	return nil
}

func (k *KeyPair) ReadPlayer(playerName string) (string, map[string]uuid.UUID, error) {
	key := map[string]*dynamodb.AttributeValue{
		"Name": {S: aws.String(playerName)},
	}

	var pd PlayerData
	err := k.Get("players", key, &pd)
	if err != nil {
		log.Printf("Error reading player data: %v", err)
		return "", nil, fmt.Errorf("player not found")
	}

	if pd.Name == "" {
		pd.Name = playerName
	}

	characterList := make(map[string]uuid.UUID)
	for name, idString := range pd.CharacterList {
		id, err := uuid.Parse(idString)
		if err != nil {
			log.Printf("Error parsing UUID for character %s: %v", name, err)
			continue
		}
		characterList[name] = id
	}

	log.Printf("Successfully read player data for %s with %d characters", pd.Name, len(characterList))
	return pd.Name, characterList, nil
}

func PlayerInput(p *Player) {
	log.Printf("Player %s input goroutine started", p.Name)

	var inputBuffer bytes.Buffer
	const maxBufferSize = 1024 // Maximum input size in bytes

	reader := bufio.NewReader(p.Connection)

	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				log.Printf("Player %s disconnected: %v", p.Name, err)
				p.PlayerError <- err
				break
			} else {
				log.Printf("Error reading from player %s: %v", p.Name, err)
				p.PlayerError <- err
				continue
			}
		}

		if p.Echo && char != '\n' && char != '\r' {
			if _, err := p.Connection.Write([]byte(string(char))); err != nil {
				log.Printf("Failed to echo character to player %s: %v", p.Name, err)
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
			log.Printf("Input buffer overflow for player %s, discarding input", p.Name)
			p.ToPlayer <- "\n\rInput too long, discarded.\n\r"
			inputBuffer.Reset()
			continue
		}

		inputBuffer.WriteRune(char)
	}

	close(p.FromPlayer)
}

func PlayerOutput(p *Player) {

	log.Printf("Player %s output goroutine started", p.Name)

	for message := range p.ToPlayer {
		// Append carriage return and newline for SSH protocol compatibility
		messageToSend := message
		if _, err := p.Connection.Write([]byte(messageToSend)); err != nil {
			log.Printf("Failed to send message to player %s: %v", p.Name, err)
			// Consider whether to continue or break based on your error handling policy
			continue
		}
	}

	// Optionally, perform any cleanup here after the channel is closed and loop exits
	log.Printf("Message channel closed for player %s", p.Name)
}

func InputLoop(c *Character) {
	log.Printf("Starting input loop for character %s", c.Name)

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
					log.Printf("Player %s issued command: %s", c.Player.Name, strings.Join(tokens, " "))
				}
				lastCommand = ""
				if !shouldQuit {
					c.Player.ToPlayer <- c.Player.Prompt
				}
			}

		case inputLine, more := <-c.Player.FromPlayer:
			if !more {
				log.Printf("Input channel closed for player %s.", c.Player.Name)
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
		log.Printf("Error saving character %s: %v", c.Name, err)
	}

	log.Printf("Input loop ended for character %s", c.Name)
}

func SelectCharacter(player *Player, server *Server) (*Character, error) {
	log.Printf("Player %s is selecting a character", player.Name)

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
				log.Printf("Error loading character %s for player %s: %v", characterName, player.Name, err)
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

		log.Printf("Character %s (ID: %s) selected and added to server character list and room", character.Name, character.ID)

		return character, nil
	}
}

func CreateCharacter(player *Player, server *Server) (*Character, error) {
	log.Printf("Player %s is creating a new character", player.Name)

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

	if server.CharacterExists[strings.ToLower(charName)] {
		return nil, fmt.Errorf("character already exists")
	}

	var selectedArchetype string
	if server.Archetypes != nil && len(server.Archetypes.Archetypes) > 0 {
		// ... (keep existing archetype selection logic)
	}

	log.Printf("Creating character with name: %s", charName)

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

	log.Printf("Added character %s (ID: %s) to player %s's character list", charName, character.ID, player.Name)

	// Save the character to the database
	err := server.Database.WriteCharacter(character)
	if err != nil {
		log.Printf("Error saving character %s to database: %v", charName, err)
		return nil, fmt.Errorf("failed to save character to database")
	}

	// Save the updated player data
	err = server.Database.WritePlayer(player)
	if err != nil {
		log.Printf("Error saving player data for %s: %v", player.Name, err)
		return nil, fmt.Errorf("failed to save player data")
	}

	log.Printf("Successfully created and saved character %s (ID: %s) for player %s", charName, character.ID, player.Name)

	return character, nil
}
