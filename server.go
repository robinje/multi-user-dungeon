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
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	Port        uint16
	Listener    net.Listener
	SSHConfig   *ssh.ServerConfig
	PlayerIndex uint32
	Players     map[uint32]*Player
	PlayerCount uint32
	RoomCount   uint32
	Mutex       sync.Mutex
	Config      Configuration
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
	s.Players = make(map[uint32]*Player) // Initialize the Players map
	s.PlayerIndex = 0
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

		uuid, _ := uuid.NewRandom()

		player := &Player{
			Name:        sshConn.User(),
			UUID:        uuid.String(),
			Index:       s.PlayerIndex,
			ToPlayer:    make(chan string),
			FromPlayer:  make(chan string),
			PlayerError: make(chan error),
			Prompt:      "Command> ",
			Connection:  channel,
			Server:      s,
		}

		// Handle SSH requests (pty-req, shell, window-change)
		go player.HandleSSHRequests(requests)

		// Initialize player
		go func(p *Player) {
			defer p.Connection.Close()

			log.Printf("Player %s connected", p.Name)

			InputLoop(p)

			s.Mutex.Lock()
			delete(s.Players, p.Index)
			s.Mutex.Unlock()
		}(player)

		s.Mutex.Lock()
		s.Players[s.PlayerIndex] = player
		s.PlayerIndex++
		s.Mutex.Unlock()
	}
}

func InputLoop(player *Player) {
	reader := bufio.NewReader(player.Connection)

	go func() {
		for msg := range player.ToPlayer {
			player.Connection.Write([]byte(msg))
		}
	}()

	player.WritePrompt()

	var inputBuffer bytes.Buffer
	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			if err != io.EOF {
				player.Connection.Write([]byte(fmt.Sprintf("Error: %v\n\r", err)))
			}
			return
		}

		// Echo the character back to the player
		player.Connection.Write([]byte(string(char)))

		// Add character to buffer
		inputBuffer.WriteRune(char)

		// Check if the character is a newline
		if char == '\n' || char == '\r' {
			inputLine := inputBuffer.String()

			// Normalize line ending to \n\r
			inputLine = strings.Replace(inputLine, "\n", "\n\r", -1)

			// Process the command
			verb, tokens, err := validateCommand(strings.TrimSpace(inputLine), valid_commands)
			if err != nil {
				player.Connection.Write([]byte(err.Error() + "\n\r"))
				player.WritePrompt()
				inputBuffer.Reset()
				continue
			}

			if executeCommand(player, verb, tokens) {
				time.Sleep(100 * time.Millisecond)
				inputBuffer.Reset()
				return
			}

			log.Printf("Player %s issued command: %s", player.Name, strings.Join(tokens, " "))
			player.WritePrompt()
			inputBuffer.Reset()
		}
	}
}

// Helper function to parse terminal dimensions from payload
func parseDims(b []byte) (width, height int) {
	width = int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	height = int(b[4])<<24 | int(b[5])<<16 | int(b[6])<<8 | int(b[7])
	return width, height
}
