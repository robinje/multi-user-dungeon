package main

type Room struct {
	RoomID      int32
	Name        string
	Description string
	Characters  map[int32]*Character
}
