package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"

	bolt "go.etcd.io/bbolt"
)

type Room struct {
	RoomID      int64
	Area        string
	Title       string
	Description string
	Exits       map[string]*Exit
	Characters  map[uint64]*Character
	Mutex       sync.Mutex
	Objects     []*Object
}

type Exit struct {
	ExitID     int64
	TargetRoom int64
	Visible    bool
	Direction  string
}

func (kp *KeyPair) LoadRooms() (map[int64]*Room, error) {
	rooms := make(map[int64]*Room)

	err := kp.db.View(func(tx *bolt.Tx) error {
		roomsBucket := tx.Bucket([]byte("Rooms"))
		if roomsBucket == nil {
			return fmt.Errorf("rooms bucket not found")
		}

		log.Printf("Using Rooms bucket: %v", roomsBucket)

		exitsBucket := tx.Bucket([]byte("Exits"))
		if exitsBucket == nil {
			return fmt.Errorf("exits bucket not found")
		}

		log.Printf("Using Exits bucket: %v", exitsBucket)

		err := roomsBucket.ForEach(func(k, v []byte) error {
			var room Room
			if err := json.Unmarshal(v, &room); err != nil {
				return fmt.Errorf("error unmarshalling room data for key %s: %w", k, err)
			}

			log.Printf("Loaded room %d: %s", room.RoomID, room.Title)

			rooms[room.RoomID] = &room
			return nil
		})
		if err != nil {
			return err
		}

		return exitsBucket.ForEach(func(k, v []byte) error {
			var exit Exit
			if err := json.Unmarshal(v, &exit); err != nil {
				return fmt.Errorf("error unmarshalling exit data for key %s: %w", k, err)
			}

			keyParts := strings.SplitN(string(k), "_", 2)
			if len(keyParts) != 2 {
				return fmt.Errorf("invalid exit key format: %s", k)
			}
			roomID, err := strconv.ParseInt(keyParts[0], 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing room ID from key %s: %w", k, err)
			}

			if room, exists := rooms[roomID]; exists {
				room.Exits[exit.Direction] = &exit
			} else {
				return fmt.Errorf("room not found for exit key %s", k)
			}
			return nil
		})
	})

	if err != nil {
		return nil, fmt.Errorf("error reading from BoltDB: %w", err)
	}

	return rooms, nil
}

func NewRoom(RoomID int64, Area string, Title string, Description string) *Room {
	room := &Room{
		RoomID:      RoomID,
		Area:        Area,
		Title:       Title,
		Description: Description,
		Exits:       make(map[string]*Exit),
		Characters:  make(map[uint64]*Character),
		Mutex:       sync.Mutex{},
		Objects:     make([]*Object, 0),
	}

	log.Printf("Created room %s with ID %d", room.Title, room.RoomID)

	return room
}

func (r *Room) AddExit(exit *Exit) {
	r.Exits[exit.Direction] = exit
}

func (r *Room) SendRoomMessage(message string) {

	for _, character := range r.Characters {
		character.Player.ToPlayer <- message

		character.Player.ToPlayer <- character.Player.Prompt

	}
}

func (r *Room) RoomInfo(character *Character) string {
	roomInfo := fmt.Sprintf("\n\r[%s]\n\r%s\n\r", ApplyColor("white", r.Title), r.Description)
	var displayExits strings.Builder

	exits := make([]string, 0)
	for direction := range r.Exits {
		exits = append(exits, direction)
	}

	sort.Strings(exits)

	if len(exits) == 0 {
		displayExits.WriteString("There are no exits.\n\r")
	} else {
		displayExits.WriteString("Obvious exits: ")
		for i, exit := range exits {
			if i > 0 {
				displayExits.WriteString(", ")
			}
			displayExits.WriteString(exit)
		}
		displayExits.WriteString("\n\r")
	}

	var charactersInRoom strings.Builder
	for _, c := range r.Characters {
		if c != character {
			charactersInRoom.WriteString(c.Name + ", ")
		}
	}
	if charactersInRoom.Len() > 0 {
		charactersInRoomStr := charactersInRoom.String()
		roomInfo += "Also here: " + charactersInRoomStr[:len(charactersInRoomStr)-2] + "\n\r"
	} else {
		roomInfo += "You are alone.\n\r"
	}

	// Display objects in the room
	if len(r.Objects) > 0 {
		roomInfo += "Objects in the room:\n\r"
		for _, obj := range r.Objects {
			roomInfo += "- " + obj.Name + "\n\r"
		}
	}

	return roomInfo + displayExits.String()
}
