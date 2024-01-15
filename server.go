package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"
	"time"

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
}

func (s *Server) StartSSHServer() error {
	privateBytes, err := ioutil.ReadFile("./server.key")
	if err != nil {
		return err
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return err
	}

	s.SSHConfig = &ssh.ServerConfig{
		NoClientAuth: true,
	}
	s.SSHConfig.AddHostKey(private)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return err
	}
	s.Listener = listener

	log.Printf("SSH server listening on port %d", s.Port)

	s.Mutex.Lock()
	s.Players = make(map[uint32]*Player) // Initialize the Players map
	s.PlayerCount = 0
	s.RoomCount = 0
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
		go s.handleChannels(chans)
	}
}

func (s *Server) handleChannels(channels <-chan ssh.NewChannel) {
	for newChannel := range channels {
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
			continue
		}

		player := &Player{
			Name:        "",
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

			p.AskForName()

			InputLoop(p)

			s.Mutex.Lock()
			s.PlayerCount--
			delete(s.Players, p.Index)
			s.Mutex.Unlock()
		}(player)

		s.Mutex.Lock()
		s.Players[s.PlayerIndex] = player
		s.PlayerIndex++
		s.PlayerCount++
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

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			player.Connection.Write([]byte(fmt.Sprintf("Error: %v\n\r", err)))
			return
		}

		verb, tokens, err := validateCommand(strings.TrimSpace(input), valid_commands)
		if err != nil {
			player.Connection.Write([]byte(err.Error() + "\n\r"))
			player.WritePrompt()
			continue
		}

		if executeCommand(player, verb, tokens) {
			time.Sleep(100 * time.Millisecond)
			return
		}

		log.Printf("Player %s issued command: %s", player.Name, strings.Join(tokens, " "))

		player.WritePrompt()
	}
}

// Helper function to parse terminal dimensions from payload
func parseDims(b []byte) (width, height int) {
	width = int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	height = int(b[4])<<24 | int(b[5])<<16 | int(b[6])<<8 | int(b[7])
	return width, height
}
