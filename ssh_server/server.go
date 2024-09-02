package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/robinje/multi-user-dungeon/core"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

func NewServer(config core.Configuration) (*core.Server, error) {

	core.Logger.Info("Initializing server...")

	// Initialize the server with the configuration
	server := &core.Server{
		Port:        config.Server.Port,
		PlayerIndex: &core.Index{},
		Config:      config,
		Context:     context.Background(),
		StartTime:   time.Now(),
		Rooms:       make(map[int64]*core.Room),
		Characters:  make(map[uuid.UUID]*core.Character),
		Balance:     config.Game.Balance,
		AutoSave:    config.Game.AutoSave,
		Health:      config.Game.StartingHealth,
		Essence:     config.Game.StartingEssence,
	}

	core.Logger.Info("Initializing database...")

	// Initialize the database
	var err error
	server.Database, err = core.NewKeyPair(config.Aws.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	// Establish the player index
	server.PlayerIndex.IndexID = 1

	// Load the character names from the database

	core.Logger.Info("Loading character names from database...")

	server.CharacterExists, err = server.Database.LoadCharacterNames()
	if err != nil {
		core.Logger.Error("Error loading character names from database", "error", err)
	}

	server.Archetypes, err = server.Database.LoadArchetypes()
	if err != nil {
		core.Logger.Error("Error loading archetypes from database", "error", err)
	}

	// Add a default room

	server.Rooms[0] = core.NewRoom(0, "The Void", "The Void", "You are in a void of nothingness. If you are here, something has gone terribly wrong.")

	// Load rooms into the server

	core.Logger.Info("Loading rooms from database...")

	server.Rooms, err = server.Database.LoadRooms()
	if err != nil {
		return nil, fmt.Errorf("failed to load rooms: %v", err)
	}

	return server, nil
}

func loadConfiguration(configFile string) (core.Configuration, error) {
	var config core.Configuration

	data, err := os.ReadFile(configFile)
	if err != nil {
		return config, fmt.Errorf("error reading config file: %w", err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return config, nil
}

func main() {
	configFile := flag.String("config", "config.yml", "Configuration file")
	flag.Parse()

	config, err := loadConfiguration(*configFile)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		return
	}

	// Initialize logging
	err = core.InitializeLogging(&config)
	if err != nil {
		fmt.Printf("Error initializing logging: %v\n", err)
		return
	}

	server, err := NewServer(config)
	if err != nil {
		core.Logger.Error("Failed to create server", "error", err)
		return
	}

	// Start sending metrics
	go func() {
		if err := core.SendMetrics(server, 1*time.Minute); err != nil {
			core.Logger.Error("Error in SendMetrics", "error", err)
		}
	}()

	// Start the auto-save routine
	go core.AutoSave(server)

	// Start the server
	if err := StartSSHServer(server); err != nil {
		core.Logger.Error("Failed to start server", "error", err)
		return
	}
}

func Authenticate(username, password string, config core.Configuration) bool {

	core.Logger.Info("Authenticating user", "username", username)

	_, err := core.SignInUser(username, password, config)
	if err != nil {
		core.Logger.Error("Authentication attempt failed for user", "username", username, "error", err)
		return false
	}
	return true
}

func StartSSHServer(server *core.Server) error {

	core.Logger.Info("Starting SSH server", "port", server.Port)

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
				core.Logger.Info("Player authenticated", "player_name", conn.User())
				return nil, nil
			}
			core.Logger.Warn("Player failed authentication", "player_name", conn.User())
			return nil, fmt.Errorf("password rejected for %q", conn.User())
		},
	}

	server.SSHConfig.AddHostKey(private)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", server.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", server.Port, err)
	}

	server.Listener = listener

	core.Logger.Info("SSH server listening", "port", server.Port)

	for {
		conn, err := server.Listener.Accept()
		if err != nil {
			core.Logger.Error("Error accepting connection", "error", err)
			continue
		}

		sshConn, chans, reqs, err := ssh.NewServerConn(conn, server.SSHConfig)
		if err != nil {
			core.Logger.Error("Failed to handshake", "error", err)
			continue
		}

		go ssh.DiscardRequests(reqs)

		go handleChannels(server, sshConn, chans)
	}
}

func handleChannels(server *core.Server, sshConn *ssh.ServerConn, channels <-chan ssh.NewChannel) {

	core.Logger.Info("New connection", "address", sshConn.RemoteAddr().String(), "user", sshConn.User())

	for newChannel := range channels {
		channel, requests, err := newChannel.Accept()
		if err != nil {
			core.Logger.Error("Could not accept channel", "error", err)
			continue
		}

		playerName := sshConn.User()
		playerIndex := server.PlayerIndex.GetID()

		// Attempt to read the player from the database
		_, characterList, err := server.Database.ReadPlayer(playerName)
		if err != nil {
			// If the player does not exist, create a new record
			if err.Error() == "player not found" {
				core.Logger.Info("Creating new player record", "player_name", playerName)
				characterList = make(map[string]uuid.UUID) // Initialize an empty character list for new players
				err = server.Database.WritePlayer(&core.Player{
					Name:          playerName,
					CharacterList: characterList,
				})
				if err != nil {
					core.Logger.Error("Error creating player record", "error", err)
					continue
				}
			} else {
				core.Logger.Error("Error reading player from database", "error", err)
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

			core.Logger.Info("Player connected", "player_name", p.Name)

			// Send welcome message
			p.ToPlayer <- fmt.Sprintf("Welcome to the game, %s!\n\r", p.Name)

			// Character Selection Dialog
			character, _ := core.SelectCharacter(p, server)

			core.InputLoop(character)

			close(player.ToPlayer)

			server.Database.WriteCharacter(character)

			server.Database.WritePlayer(player)

			core.Logger.Info("Player disconnected", "player_name", p.Name)
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

	core.Logger.Info("Handling SSH requests for player", "player_name", player.Name)

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
