package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Port        uint16       // Port to listen on
	Listener    net.Listener // Listener for the server
	PlayerIndex uint32
	Players     map[uint32]*Player
	PlayerCount uint32
	RoomCount   uint32
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
	s.PlayerIndex = 0
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
		Index:       s.PlayerIndex,
		ToPlayer:    make(chan string),
		FromPlayer:  make(chan string),
		PlayerError: make(chan error),
		Prompt:      "Command> ",
		Connection:  conn,
		Server:      s,
	}

	// Start goroutine for handling player messages immediately
	go func() {
		for msg := range player.ToPlayer {
			conn.Write([]byte(msg))
		}
	}()

	s.Mutex.Lock()
	s.Players[s.PlayerIndex] = player
	s.PlayerIndex++
	s.PlayerCount++
	s.Mutex.Unlock()

	// Ask the player for their name
	player.AskForName()

	InputLoop(player)

	// Cleanup when the player disconnects
	s.Mutex.Lock()
	s.PlayerCount--
	delete(s.Players, player.Index)
	s.Mutex.Unlock()
}

func InputLoop(player *Player) {
	reader := bufio.NewReader(player.Connection)

	// Goroutine for handling player messages
	go func() {
		for msg := range player.ToPlayer {
			player.Connection.Write([]byte(msg))
		}
	}()

	// Initially write the prompt
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

		// Write the prompt after processing the command
		player.WritePrompt()
	}
}
