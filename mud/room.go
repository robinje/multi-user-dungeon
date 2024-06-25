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
	Items       []*Item
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

		itemsBucket := tx.Bucket([]byte("Items"))
		if itemsBucket == nil {
			log.Printf("Items bucket not found, no items will be loaded")
		} else {
			log.Printf("Using Items bucket: %v", itemsBucket)
		}

		// Load rooms
		err := roomsBucket.ForEach(func(k, v []byte) error {
			var roomData struct {
				Room
				ItemIndices []uint64 `json:"items"`
			}
			if err := json.Unmarshal(v, &roomData); err != nil {
				return fmt.Errorf("error unmarshalling room data for key %s: %w", k, err)
			}

			room := &roomData.Room
			room.Items = make([]*Item, 0, len(roomData.ItemIndices))
			rooms[room.RoomID] = room

			// Load items for this room
			for _, itemIndex := range roomData.ItemIndices {
				item, err := kp.LoadItem(itemIndex, false)
				if err != nil {
					log.Printf("Error loading item %d for room %d: %v", itemIndex, room.RoomID, err)
					continue
				}
				room.Items = append(room.Items, item)
			}

			log.Printf("Loaded room %d: %s with %d items", room.RoomID, room.Title, len(room.Items))
			return nil
		})
		if err != nil {
			return err
		}

		// Load exits
		err = exitsBucket.ForEach(func(k, v []byte) error {
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
		if err != nil {
			return err
		}

		return nil
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
		Items:       make([]*Item, 0),
	}

	log.Printf("Created room %s with ID %d", room.Title, room.RoomID)

	return room
}

func (r *Room) AddExit(exit *Exit) {
	r.Exits[exit.Direction] = exit
}

func (c *Character) Move(direction string) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if c.Room == nil {
		c.Player.ToPlayer <- "You are not in any room to move from.\n\r"
		return
	}

	log.Printf("Player %s is moving %s", c.Name, direction)

	selectedExit, exists := c.Room.Exits[direction]
	if !exists {
		c.Player.ToPlayer <- "You cannot go that way.\n\r"
		return
	}

	newRoom, exists := c.Server.Rooms[selectedExit.TargetRoom]
	if !exists {
		c.Player.ToPlayer <- "The path leads nowhere.\n\r"
		return
	}

	// Safely remove the character from the old room
	oldRoom := c.Room
	oldRoom.Mutex.Lock()
	delete(oldRoom.Characters, c.Index)
	oldRoom.Mutex.Unlock()
	oldRoom.SendRoomMessage(fmt.Sprintf("\n\r%s has left going %s.\n\r", c.Name, direction))

	// Update character's room
	c.Room = newRoom

	newRoom.SendRoomMessage(fmt.Sprintf("\n\r%s has arrived.\n\r", c.Name))

	// Ensure the Characters map in the new room is initialized
	newRoom.Mutex.Lock()
	if newRoom.Characters == nil {
		newRoom.Characters = make(map[uint64]*Character)
	}
	newRoom.Characters[c.Index] = c
	newRoom.Mutex.Unlock()

	executeLookCommand(c, []string{})
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
	if len(r.Items) > 0 {
		roomInfo += "Items in the room:\n\r"
		for _, obj := range r.Items {
			roomInfo += "- " + obj.Name + "\n\r"
		}
	}

	return roomInfo + displayExits.String()
}

func (r *Room) findItemByPosition(name string, position int) *Item {
	count := 0
	for _, item := range r.Items {
		if strings.EqualFold(item.Name, name) {
			count++
			if count == position {
				return item
			}
		}
	}
	return nil
}

func (r *Room) removeItem(item *Item) {
	for i, roomItem := range r.Items {
		if roomItem == item {
			r.Items = append(r.Items[:i], r.Items[i+1:]...)
			return
		}
	}
}
