package main

import (
	"bufio"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Player struct {
	Index         uint32
	Name          string
	ToPlayer      chan string
	FromPlayer    chan string
	PlayerError   chan error
	Prompt        string
	Connection    ssh.Channel // Changed from net.Conn to ssh.Channel
	Server        *Server
	ConsoleWidth  int
	ConsoleHeight int
}

// AskForName prompts the player for their name and sets it in the Player struct
func (p *Player) AskForName() {
	p.ToPlayer <- "Enter your name: "
	reader := bufio.NewReader(p.Connection)
	name, _ := reader.ReadString('\n')
	p.Name = strings.TrimSpace(name)
}

// WritePrompt sends the command prompt to the player
func (p *Player) WritePrompt() {
	p.Connection.Write([]byte(p.Prompt))
}

// HandleSSHRequests handles SSH requests from the client
func (p *Player) HandleSSHRequests(requests <-chan *ssh.Request) {
	for req := range requests {
		switch req.Type {
		case "shell":
			req.Reply(true, nil)
		case "pty-req":
			termLen := req.Payload[3]
			w, h := parseDims(req.Payload[termLen+4:])
			p.ConsoleWidth, p.ConsoleHeight = w, h
			req.Reply(true, nil)
		case "window-change":
			w, h := parseDims(req.Payload)
			p.ConsoleWidth, p.ConsoleHeight = w, h
		}
	}
}
