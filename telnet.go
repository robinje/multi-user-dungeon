package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

type Server struct {
	Port        uint16       // Port to listen on
	Listener    net.Listener // Listener for the server
	Players     []*Player    // List of players connected to the server
	PlayerCount int32
	RoomCount   int32
	Mutex       sync.Mutex
}

func (s *Server) StartTelnetServer() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return err
	}
	s.Listener = listener

	log.Printf("Telnet server listening on port %d", s.Port)

	s.Mutex.Lock()
	s.PlayerCount = 0
	s.RoomCount = 0
	s.Mutex.Unlock()

	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Create a new Player instance
	player := &Player{
		Name:        "",
		ToPlayer:    make(chan string),
		FromPlayer:  make(chan string),
		PlayerError: make(chan error),
		Prompt:      "Command> ",
	}

	s.Mutex.Lock()
	s.Players = append(s.Players, player)
	s.PlayerCount++
	s.Mutex.Unlock()

	s.InputLoop(conn, player)

	// Cleanup when the player disconnects
	s.Mutex.Lock()
	s.PlayerCount--
	s.Mutex.Unlock()
}

func (s *Server) InputLoop(conn net.Conn, player *Player) {
	reader := bufio.NewReader(conn)

	// Goroutine for handling player messages
	go func() {
		for msg := range player.ToPlayer {
			conn.Write([]byte(msg))
			conn.Write([]byte(player.Prompt)) // Write the prompt after sending a message
		}
	}()

	// Initially write the prompt
	conn.Write([]byte(player.Prompt))

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			player.PlayerError <- err
			return
		}

		verb, tokens, err := validateCommand(strings.TrimSpace(input), valid_commands)
		if err != nil {
			player.ToPlayer <- err.Error() + "\n\r"
			continue
		}

		if !executeCommand(player, verb, tokens) {
			return
		}

		log.Printf("Player %s issued command: %s", player.Name, strings.Join(tokens, " "))
	}
}
