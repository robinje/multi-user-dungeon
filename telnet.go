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

	// Set the Player and Room counts to zero
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

	s.Mutex.Lock()
	s.PlayerCount++
	s.Mutex.Unlock()

	// Start the player input loop
	s.InputLoop(conn)

	// Player input loop has exited, so player is no longer connected.
	if s.PlayerCount > 0 {
		s.Mutex.Lock()
		s.PlayerCount--
		s.Mutex.Unlock()
	}
}

func (s *Server) InputLoop(conn net.Conn) {
    reader := bufio.NewReader(conn)

    for {
        _, err := conn.Write([]byte("Command> ")) // Prompt for input
        if err != nil {
            log.Printf("Error writing to connection: %v", err)
            return
        }

        input, err := reader.ReadString('\n')
        if err != nil {
            log.Printf("Error reading from connection: %v", err)
            return // Exit the loop and close the connection
        }

        // Process the input
        input = strings.TrimSpace(input)

        // Validate the command
        tokens, err := validateCommand(input, valid_commands)
        if err != nil {
            conn.Write([]byte(err.Error() + "\n")) // Send error message back to the player
            continue
        }

        // Command is valid; process further as needed
        log.Printf("Valid command received: %s", strings.Join(tokens, " "))
    }
}
