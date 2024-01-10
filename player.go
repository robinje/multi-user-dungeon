package main

type Player struct {
	Name        string
	ToPlayer    chan string
	FromPlayer  chan string
	PlayerError chan error
	Prompt      string
}
