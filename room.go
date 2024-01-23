package main

type Room struct {
	RoomID      int32
	Area        string
	Title       string
	Description string
	Characters  map[int32]*Character
}
