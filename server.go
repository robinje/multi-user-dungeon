package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type Server struct {
	Port        uint16
	Listener    net.Listener
	SSHConfig   *ssh.ServerConfig
	Players     map[uint64]*Player
	PlayerCount uint64
	Mutex       sync.Mutex
	Config      Configuration
	StartTime   time.Time
	Rooms       map[int64]*Room
	Database    *KeyPair
	PlayerIndex *Index
}

type Index struct {
	IndexID uint64
	mu      sync.Mutex
}

func (i *Index) GetID() uint64 {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.IndexID++
	return i.IndexID
}

func (i *Index) SetID(id uint64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if id > i.IndexID {
		i.IndexID = id
	}
}

func NewServer(config Configuration) (*Server, error) {
	// Initialize the server with the configuration
	server := &Server{
		Port:        config.Port,
		Players:     make(map[uint64]*Player),
		Config:      config,
		StartTime:   time.Now(),
		Rooms:       make(map[int64]*Room),
		PlayerIndex: &Index{},
	}

	log.Printf("Initializing database...")

	// Initialize the database
	var err error
	server.Database, err = NewKeyPair(config.DataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	// Establish the player index
	server.PlayerIndex.IndexID = 1

	// Add a default room

	server.Rooms[0] = NewRoom(0, "The Void", "The Void", "You are in a void of nothingness. If you are here, something has gone terribly wrong.")

	// Load rooms into the server

	log.Printf("Loading rooms from database...")

	server.Rooms, err = server.Database.LoadRooms()
	if err != nil {
		return nil, fmt.Errorf("failed to load rooms: %v", err)
	}

	return server, nil
}

func (s *Server) authenticateWithCognito(username string, password string) bool {
	_, err := SignInUser(username, password, s.Config)
	if err != nil {
		log.Printf("Authentication failed for user %s: %v", username, err)
		return false
	}
	return true
}

func (s *Server) StartSSHServer() error {
	privateBytes, err := os.ReadFile("./server.key")
	if err != nil {
		return fmt.Errorf("failed to read private key: %v", err)
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	s.SSHConfig = &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			authenticated := s.authenticateWithCognito(conn.User(), string(password))
			if authenticated {
				log.Printf("Player %s authenticated", conn.User())
				return nil, nil
			}
			log.Printf("Player %s failed authentication", conn.User())
			return nil, fmt.Errorf("password rejected for %q", conn.User())
		},
	}
	s.SSHConfig.AddHostKey(private)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", s.Port, err)
	}
	s.Listener = listener

	log.Printf("SSH server listening on port %d", s.Port)

	s.Mutex.Lock()
	s.Players = make(map[uint64]*Player) // Initialize the Players map
	s.Mutex.Unlock()

	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.SSHConfig)
		if err != nil {
			log.Printf("Failed to handshake: %v", err)
			continue
		}

		go ssh.DiscardRequests(reqs)
		go s.handleChannels(sshConn, chans)
	}
}

func (s *Server) handleChannels(sshConn *ssh.ServerConn, channels <-chan ssh.NewChannel) {
	log.Printf("New connection from %s (%s)", sshConn.User(), sshConn.RemoteAddr())

	for newChannel := range channels {
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
			continue
		}

		playerName := sshConn.User()
		playerIndex := s.PlayerIndex.GetID()

		// Attempt to read the player from the database
		_, characterList, err := s.Database.ReadPlayer(playerName)
		if err != nil {
			// If the player does not exist, create a new record
			if err.Error() == "player not found" {
				log.Printf("Creating new player record for %s", playerName)
				characterList = make(map[string]uint64) // Initialize an empty character list for new players
				err = s.Database.WritePlayer(&Player{
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
		player := &Player{
			Name:          playerName,
			Index:         playerIndex,
			ToPlayer:      make(chan string),
			FromPlayer:    make(chan string),
			PlayerError:   make(chan error),
			Prompt:        "> ",
			Connection:    channel,
			Server:        s,
			CharacterList: characterList,
		}

		// Handle SSH requests (pty-req, shell, window-change)
		go player.HandleSSHRequests(requests)

		// Start the goroutine responsible for handling outgoing messages to the player
		player.StartMessageHandler()

		// Initialize player
		go func(p *Player) {
			defer p.Connection.Close()

			log.Printf("Player %s connected", p.Name)

			// Send welcome message
			p.SendMessage(fmt.Sprintf("Welcome to the game, %s!", p.Name))

			// Character Selection Dialog
			character, _ := s.SelectCharacter(p)

			character.InputLoop()

			close(player.ToPlayer)

			s.WriteCharacter(character)

			s.Mutex.Lock()
			delete(s.Players, p.Index)
			s.Mutex.Unlock()
		}(player)

		s.Mutex.Lock()
		s.Players[player.Index] = player
		s.Mutex.Unlock()
	}
}

// Helper function to parse terminal dimensions from payload
func parseDims(b []byte) (width, height int) {
	width = int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	height = int(b[4])<<24 | int(b[5])<<16 | int(b[6])<<8 | int(b[7])
	return width, height
}
