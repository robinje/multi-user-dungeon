package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"

	bolt "go.etcd.io/bbolt"
)

func (k *KeyPair) WritePlayer(player *Player) error {
	// Create a PlayerData instance containing only the data to be serialized
	pd := PlayerData{
		Name:          player.Name,
		CharacterList: player.CharacterList,
	}

	// Serialize the PlayerData struct to JSON
	playerData, err := json.Marshal(pd)
	if err != nil {
		return fmt.Errorf("error marshalling player data: %w", err)
	}

	// Use the player's Name as the key to store the serialized data
	err = k.Put("Players", []byte(player.Name), playerData)
	if err != nil {
		return fmt.Errorf("error storing player data: %w", err)
	}

	log.Printf("Successfully wrote player data for %s with %d characters", player.Name, len(player.CharacterList))
	return nil
}

func (k *KeyPair) ReadPlayer(playerName string) (string, map[string]uint64, error) {
	playerData, err := k.Get("Players", []byte(playerName))
	if err != nil {
		if err == bolt.ErrBucketNotFound {
			log.Println("Player bucket not found")
			return "", nil, fmt.Errorf("player not found")
		}
		log.Printf("Error reading player data: %v", err)
		return "", nil, fmt.Errorf("database read failed: %w", err)
	}

	if playerData == nil {
		log.Printf("Player %s not found", playerName)
		return "", nil, fmt.Errorf("player not found")
	}

	var pd PlayerData
	if err := json.Unmarshal(playerData, &pd); err != nil {
		log.Printf("Error unmarshalling player data: %v", err)
		return "", nil, fmt.Errorf("unmarshal player data: %w", err)
	}

	// If Name is empty in the database, use the playerName parameter
	if pd.Name == "" {
		pd.Name = playerName
	}

	// Ensure CharacterList is initialized
	if pd.CharacterList == nil {
		pd.CharacterList = make(map[string]uint64)
	}

	log.Printf("Successfully read player data for %s with %d characters", pd.Name, len(pd.CharacterList))
	return pd.Name, pd.CharacterList, nil
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
	ExecuteLookCommand(c, []string{}) // Adjusted to include the tokens parameter

	// Send initial prompt to player
	c.Player.ToPlayer <- c.Player.Prompt

	for {
		// Wait for input from the player. This blocks until input is received.
		inputLine, more := <-c.Player.FromPlayer
		if !more {
			// If the channel is closed, stop the input loop.
			log.Printf("Input channel closed for player %s.", c.Player.Name)
			return
		}

		// Normalize line ending to \n\r for consistency
		inputLine = strings.Replace(inputLine, "\n", "\n\r", -1)

		// Process the command
		verb, tokens, err := ValidateCommand(strings.TrimSpace(inputLine))
		if err != nil {
			c.Player.ToPlayer <- err.Error() + "\n\r"
			c.Player.ToPlayer <- c.Player.Prompt
			continue
		}

		// Execute the command
		if ExecuteCommand(c, verb, tokens) {
			// If command execution indicates to exit (or similar action), break the loop
			// Note: Adjust logic as per your executeCommand's design to handle such conditions
			break
		}

		// Log the command execution
		log.Printf("Player %s issued command: %s", c.Player.Name, strings.Join(tokens, " "))

		// Prompt for the next command
		c.Player.ToPlayer <- c.Player.Prompt
	}

	// Close the player's input channel
	close(c.Player.FromPlayer)

	// Remove the character from the room

	c.Room.Mutex.Lock()
	delete(c.Room.Characters, c.Index)
	c.Room.Mutex.Unlock()

	// Remove the character from the server's active characters
	c.Server.Mutex.Lock()
	delete(c.Server.Characters, c.Name)
	c.Server.Mutex.Unlock()

	// Save the character to the database
	err := c.Server.Database.WriteCharacter(c)
	if err != nil {
		log.Printf("Error saving character %s: %v", c.Name, err)
	}
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
			characterIndex := player.CharacterList[characterName]
			character, err = server.Database.LoadCharacter(characterIndex, player, server)
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
		server.Characters[character.Name] = character
		server.Mutex.Unlock()

		log.Printf("Character %s (ID: %d) selected and added to server character list", character.Name, character.Index)

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

	log.Printf("Creating character with name: %s", charName)

	room, ok := server.Rooms[1]
	if !ok {
		room, ok = server.Rooms[0]
		if !ok {
			return nil, fmt.Errorf("no starting room found")
		}
	}

	character := server.NewCharacter(charName, player, room, selectedArchetype)

	// Add the new character to the player's character list
	player.Mutex.Lock()
	if player.CharacterList == nil {
		player.CharacterList = make(map[string]uint64)
	}
	player.CharacterList[charName] = character.Index
	player.Mutex.Unlock()

	log.Printf("Added character %s (ID: %d) to player %s's character list", charName, character.Index, player.Name)

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

	log.Printf("Successfully created and saved character %s (ID: %d) for player %s", charName, character.Index, player.Name)

	return character, nil
}
