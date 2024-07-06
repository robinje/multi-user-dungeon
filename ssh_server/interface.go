package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/robinje/multi-user-dungeon/core"
	"golang.org/x/crypto/ssh"
)

func Authenticate(username, password string, config core.Configuration) bool {

	log.Printf("Authenticating user %s", username)

	_, err := core.SignInUser(username, password, config)
	if err != nil {
		log.Printf("Authentication attempt failed for user %s: %v", username, err)
		return false
	}
	return true
}

func StartSSHServer(server *core.Server) error {

	log.Printf("Starting SSH server on port %d", server.Port)

	// Read the private key from disk
	privateBytes, err := os.ReadFile("./server.key")
	if err != nil {
		return fmt.Errorf("failed to read private key: %v", err)
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	server.SSHConfig = &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			// Authenticate the player
			authenticated := Authenticate(conn.User(), string(password), server.Config)
			if authenticated {
				log.Printf("Player %s authenticated", conn.User())
				return nil, nil
			}
			log.Printf("Player %s failed authentication", conn.User())
			return nil, fmt.Errorf("password rejected for %q", conn.User())
		},
	}

	server.SSHConfig.AddHostKey(private)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", server.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", server.Port, err)
	}

	server.Listener = listener

	log.Printf("SSH server listening on port %d", server.Port)

	for {
		conn, err := server.Listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		sshConn, chans, reqs, err := ssh.NewServerConn(conn, server.SSHConfig)
		if err != nil {
			log.Printf("Failed to handshake: %v", err)
			continue
		}

		go ssh.DiscardRequests(reqs)

		go handleChannels(server, sshConn, chans)
	}
}

func handleChannels(server *core.Server, sshConn *ssh.ServerConn, channels <-chan ssh.NewChannel) {

	log.Printf("New connection from %s (%s)", sshConn.User(), sshConn.RemoteAddr())

	for newChannel := range channels {
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
			continue
		}

		playerName := sshConn.User()
		playerIndex := server.PlayerIndex.GetID()

		// Attempt to read the player from the database
		_, characterList, err := server.Database.ReadPlayer(playerName)
		if err != nil {
			// If the player does not exist, create a new record
			if err.Error() == "player not found" {
				log.Printf("Creating new player record for %s", playerName)
				characterList = make(map[string]uint64) // Initialize an empty character list for new players
				err = server.Database.WritePlayer(&core.Player{
					Name:          playerName,
					CharacterList: characterList,
				})
				if err != nil {
					log.Printf("Error creating player record: %v", err)
					continue
				}
			} else {
				log.Printf("Error reading player from database: %v", err)
				continue
			}
		}

		// Create the Player struct with data from the database or as a new player
		player := &core.Player{
			Name:          playerName,
			Index:         playerIndex,
			ToPlayer:      make(chan string),
			FromPlayer:    make(chan string),
			PlayerError:   make(chan error),
			Echo:          true,
			Prompt:        "> ",
			Connection:    channel,
			Server:        server,
			CharacterList: characterList,
		}

		// Handle SSH requests (pty-req, shell, window-change)
		go HandleSSHRequests(player, requests)

		// Start the goroutine responsible for player I/O
		go PlayerInput(player)
		go PlayerOutput(player)

		// Initialize player
		go func(p *core.Player) {
			defer p.Connection.Close()

			log.Printf("Player %s connected", p.Name)

			// Send welcome message
			p.ToPlayer <- fmt.Sprintf("Welcome to the game, %s!\n\r", p.Name)

			// Character Selection Dialog
			character, _ := core.SelectCharacter(p, server)

			InputLoop(character)

			close(player.ToPlayer)

			server.Database.WriteCharacter(character)

			log.Printf("Player %s disconnected", p.Name)
			player = nil

		}(player)

	}
}

// Helper function to parse terminal dimensions from payload
func parseDims(b []byte) (width, height int) {
	width = int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	height = int(b[4])<<24 | int(b[5])<<16 | int(b[6])<<8 | int(b[7])
	return width, height
}

// HandleSSHRequests handles SSH requests from the client
func HandleSSHRequests(player *core.Player, requests <-chan *ssh.Request) {

	log.Printf("Handling SSH requests for player %s", player.Name)

	for req := range requests {
		switch req.Type {
		case "shell":
			req.Reply(true, nil)
		case "pty-req":
			termLen := req.Payload[3]
			w, h := parseDims(req.Payload[termLen+4:])
			player.ConsoleWidth, player.ConsoleHeight = w, h
			req.Reply(true, nil)
		case "window-change":
			w, h := parseDims(req.Payload)
			player.ConsoleWidth, player.ConsoleHeight = w, h
		}
	}
}

func PlayerInput(p *core.Player) {

	log.Printf("Player %s input goroutine started", p.Name)

	var inputBuffer bytes.Buffer

	reader := bufio.NewReader(p.Connection)

	for {
		char, _, err := reader.ReadRune() // Read one rune (character) at a time from the buffered reader
		if err != nil {
			if err == io.EOF {
				// Handle EOF to indicate client disconnect gracefully
				log.Printf("Player %s disconnected: %v", p.Name, err)
				p.PlayerError <- err
				break // Exit the loop on EOF
			} else {
				// Log and handle other errors without breaking the loop
				log.Printf("Error reading from player %s: %v", p.Name, err)
				p.PlayerError <- err
				continue
			}
		}

		// Echo the character back to the player if Echo is true
		// Ensure we do not echo back newline characters, maintaining input cleanliness
		if p.Echo && char != '\n' && char != '\r' {
			if _, err := p.Connection.Write([]byte(string(char))); err != nil {
				log.Printf("Failed to echo character to player %s: %v", p.Name, err)
			}
		}

		// Check if the character is a newline, indicating the end of input
		if char == '\n' || char == '\r' {
			// Trim the newline character and send the input through the FromPlayer channel
			// This assumes that the inputBuffer contains the input line up to the newline character
			if inputBuffer.Len() > 0 { // Ensure we have something to send
				p.FromPlayer <- inputBuffer.String()
				inputBuffer.Reset() // Clear the buffer for the next line of input
			}
			continue
		}

		// Add character to the buffer for accumulating the line
		inputBuffer.WriteRune(char)
	}

	// Close the channel to signify no more input will be processed
	close(p.FromPlayer)
}

func PlayerOutput(p *core.Player) {

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

func InputLoop(c *core.Character) {

	log.Printf("Starting input loop for character %s", c.Name)

	// Initially execute the look command with no additional tokens
	core.ExecuteLookCommand(c, []string{}) // Adjusted to include the tokens parameter

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
		verb, tokens, err := core.ValidateCommand(strings.TrimSpace(inputLine))
		if err != nil {
			c.Player.ToPlayer <- err.Error() + "\n\r"
			c.Player.ToPlayer <- c.Player.Prompt
			continue
		}

		// Execute the command
		if core.ExecuteCommand(c, verb, tokens) {
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
