package main

import (
	"golang.org/x/crypto/ssh"
)

type Player struct {
	PlayerID      int32
	Index         uint32
	Name          string
	ToPlayer      chan string
	FromPlayer    chan string
	PlayerError   chan error
	Prompt        string
	Connection    ssh.Channel
	Server        *Server
	ConsoleWidth  int
	ConsoleHeight int
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
