package main

import (
	"bufio"
	"net"
	"strings"
)

type Player struct {
	Index       uint32
	Name        string
	ToPlayer    chan string
	FromPlayer  chan string
	PlayerError chan error
	Prompt      string
	Connection  net.Conn
}

// AskForName prompts the player for their name and sets it in the Player struct
func (p *Player) AskForName() {
	p.ToPlayer <- "Enter your name: "
	reader := bufio.NewReader(p.Connection)
	name, _ := reader.ReadString('\n')
	p.Name = strings.TrimSpace(name)
}
