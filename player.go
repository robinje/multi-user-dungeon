package main

import (
	"bufio"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Player struct {
	Index       uint32
	Name        string
	ToPlayer    chan string
	FromPlayer  chan string
	PlayerError chan error
	Prompt      string
	Connection  ssh.Channel // Changed from net.Conn to ssh.Channel
	Server      *Server
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
