package main

type Room struct {
	RoomID      int32
	Area        string
	Title       string
	Description string
	Exits       map[string]*Exit
	Characters  map[int32]*Character
}

type Exit struct {
	ExitID     int32
	TargetRoom int32
	Visibile   bool
	Direction  string
}
