package main

import (
	"fmt"
	"log"
	"net"
	"os"

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
		go core.PlayerInput(player)
		go core.PlayerOutput(player)

		// Initialize player
		go func(p *core.Player) {
			defer p.Connection.Close()

			log.Printf("Player %s connected", p.Name)

			// Send welcome message
			p.ToPlayer <- fmt.Sprintf("Welcome to the game, %s!\n\r", p.Name)

			// Character Selection Dialog
			character, _ := core.SelectCharacter(p, server)

			core.InputLoop(character)

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
