package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	bolt "go.etcd.io/bbolt"
)

type Room struct {
	RoomID      int64
	Area        string
	Title       string
	Description string
	Exits       map[string]*Exit
	Characters  map[uint64]*Character
}

type Exit struct {
	ExitID     int64
	TargetRoom int64
	Visibile   bool
	Direction  string
}

func (kp *KeyPair) LoadRooms() (map[int64]*Room, error) {
	rooms := make(map[int64]*Room)

	err := kp.db.View(func(tx *bolt.Tx) error {
		roomsBucket := tx.Bucket([]byte("Rooms"))
		if roomsBucket == nil {
			return fmt.Errorf("Rooms bucket not found")
		}

		log.Printf("Using Rooms bucket: %v", roomsBucket)

		exitsBucket := tx.Bucket([]byte("Exits"))
		if exitsBucket == nil {
			return fmt.Errorf("Exits bucket not found")
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
	}

	log.Printf("Created room %s with ID %d", room.Title, room.RoomID)

	return room
}

func (r *Room) AddExit(exit *Exit) {
	r.Exits[exit.Direction] = exit
}

func (r *Room) SendRoomMessage(message string) {

	for _, character := range r.Characters {
		character.SendMessage(message)
	}
}

func (r *Room) RoomInfo(character *Character) string {
	roomInfo := fmt.Sprintf("\n\r[%s]\n\r%s\n\r", r.Title, r.Description)

	var displayExits strings.Builder

	exits := make([]string, 0, len(r.Exits))
	for _, exit := range r.Exits {
		exits = append(exits, exit.Direction)
	}

	switch len(exits) {
	case 0:
		displayExits.WriteString("There are no exits.\n\r")
	case 1:
		displayExits.WriteString(fmt.Sprintf("Obvious exit: %s\n\r", exits[0]))
	case 2:
		displayExits.WriteString(fmt.Sprintf("Obvious exits: %s and %s\n\r", exits[0], exits[1]))
	default:
		displayExits.WriteString("Obvious exits: ")
		for i, exit := range exits[:len(exits)-1] {
			if i > 0 {
				displayExits.WriteString(", ")
			}
			displayExits.WriteString(exit)
		}
		displayExits.WriteString(", and " + exits[len(exits)-1] + "\n\r")
	}

	var charactersInRoom strings.Builder
	otherCharacters := make([]string, 0, len(r.Characters)-1)
	for _, c := range r.Characters {
		if c != character {
			otherCharacters = append(otherCharacters, c.Name)
		}
	}

	switch len(otherCharacters) {
	case 0:
		charactersInRoom.WriteString("You are alone.\n\r")
	case 1:
		charactersInRoom.WriteString(fmt.Sprintf("Also here: %s\n\r", otherCharacters[0]))
	case 2:
		charactersInRoom.WriteString(fmt.Sprintf("Also here: %s and %s\n\r", otherCharacters[0], otherCharacters[1]))
	default:
		charactersInRoom.WriteString("Also here: ")
		for i, name := range otherCharacters[:len(otherCharacters)-1] {
			if i > 0 {
				charactersInRoom.WriteString(", ")
			}
			charactersInRoom.WriteString(name)
		}
		charactersInRoom.WriteString(", and " + otherCharacters[len(otherCharacters)-1])
	}

	roomDescription := fmt.Sprintf("%s%s%s", roomInfo, displayExits.String(), charactersInRoom.String())

	return roomDescription
}
