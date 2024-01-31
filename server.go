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
	Port           uint16
	Listener       net.Listener
	SSHConfig      *ssh.ServerConfig
	Players        map[uint64]*Player
	PlayerCount    uint64
	Mutex          sync.Mutex
	Config         Configuration
	StartTime      time.Time
	Rooms          map[int64]*Room
	Database       *DataBase
	PlayerIndex    *Index
	CharacterIndex *Index
	ExitIndex      *Index
	RoomIndex      *Index
	ObjectIndex    *Index
}

func NewServer(config Configuration) (*Server, error) {
	// Initialize the server with the configuration
	server := &Server{
		Port:           config.Port,
		Players:        make(map[uint64]*Player),
		Config:         config,
		StartTime:      time.Now(),
		Rooms:          make(map[int64]*Room),
		PlayerIndex:    &Index{},
		CharacterIndex: &Index{},
		ExitIndex:      &Index{},
		RoomIndex:      &Index{},
		ObjectIndex:    &Index{},
	}

	// Initialize the database
	database := &DataBase{
		File: config.DataFile,
	}
	if err := database.Open(); err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	server.Database = database

	server.PlayerIndex.IndexID = 1
	server.CharacterIndex.IndexID = 1
	server.ObjectIndex.IndexID = 1
	server.RoomIndex.IndexID = 100
	server.ExitIndex.IndexID = 100

	// Load rooms into the server
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

		player := &Player{
			Name:        sshConn.User(),
			Index:       s.PlayerIndex.GetID(),
			ToPlayer:    make(chan string),
			FromPlayer:  make(chan string),
			PlayerError: make(chan error),
			Prompt:      "> ",
			Connection:  channel,
			Server:      s,
		}

		// Handle SSH requests (pty-req, shell, window-change)
		go player.HandleSSHRequests(requests)

		// Initialize player
		go func(p *Player) {
			defer p.Connection.Close()

			log.Printf("Player %s connected", p.Name)

			// Send welcome message

			p.Connection.Write([]byte(fmt.Sprintf("Welcome to the game, %s!\n\r", p.Name)))

			// Charater Selection Dialog

			character, _ := s.CreateCharacter(p)

			character.InputLoop()

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
