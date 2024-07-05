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
	Items       map[string]*Item // Change to map for efficient lookups
}

type Exit struct {
	ExitID     int64
	TargetRoom int64
	Visible    bool
	Direction  string
}

func (k *KeyPair) LoadRooms() (map[int64]*Room, error) {
	rooms := make(map[int64]*Room)

	err := k.db.View(func(tx *bolt.Tx) error {
		roomsBucket := tx.Bucket([]byte("Rooms"))
		if roomsBucket == nil {
			return fmt.Errorf("rooms bucket not found")
		}

		itemsBucket := tx.Bucket([]byte("Items"))
		if itemsBucket == nil {
			return fmt.Errorf("items bucket not found")
		}

		// Load all rooms
		err := roomsBucket.ForEach(func(k, v []byte) error {
			var roomData struct {
				RoomID      int64            `json:"RoomID"`
				Area        string           `json:"Area"`
				Title       string           `json:"Title"`
				Description string           `json:"Description"`
				Exits       map[string]*Exit `json:"Exits"`
				ItemIDs     []string         `json:"ItemIDs"` // List of item IDs in the room
			}
			if err := json.Unmarshal(v, &roomData); err != nil {
				return fmt.Errorf("error unmarshalling room data for key %s: %w", k, err)
			}

			room := &Room{
				RoomID:      roomData.RoomID,
				Area:        roomData.Area,
				Title:       roomData.Title,
				Description: roomData.Description,
				Exits:       make(map[string]*Exit),
				Characters:  make(map[uint64]*Character),
				Items:       make(map[string]*Item),
				Mutex:       sync.Mutex{},
			}

			// Load exits
			for direction, exitData := range roomData.Exits {
				exit := &Exit{
					ExitID:     exitData.ExitID,
					TargetRoom: exitData.TargetRoom,
					Visible:    exitData.Visible,
					Direction:  exitData.Direction,
				}
				room.Exits[direction] = exit
			}

			// Load items for this room
			for _, itemID := range roomData.ItemIDs {
				itemData := itemsBucket.Get([]byte(itemID))
				if itemData == nil {
					log.Printf("Item %s not found for room %d", itemID, room.RoomID)
					continue
				}

				var item Item
				if err := json.Unmarshal(itemData, &item); err != nil {
					log.Printf("Error unmarshalling item %s: %v", itemID, err)
					continue
				}

				room.Items[itemID] = &item
			}

			rooms[room.RoomID] = room
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

	// Log room information
	for _, room := range rooms {
		log.Printf("Loaded room %d: %s with %d exits and %d items",
			room.RoomID, room.Title, len(room.Exits), len(room.Items))
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
		Items:       make(map[string]*Item),
		Mutex:       sync.Mutex{},
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
		c.Player.ToPlayer <- "\n\rYou are not in any room to move from.\n\r"
		return
	}

	log.Printf("Player %s is moving %s", c.Name, direction)

	selectedExit, exists := c.Room.Exits[direction]
	if !exists {
		c.Player.ToPlayer <- "\n\rYou cannot go that way.\n\r"
		return
	}

	newRoom, exists := c.Server.Rooms[selectedExit.TargetRoom]
	if !exists {
		c.Player.ToPlayer <- "\n\rThe path leads nowhere.\n\r"
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

func (kp *KeyPair) WriteRoom(room *Room) error {
	// Create a serializable version of the room data
	roomData := struct {
		RoomID      int64            `json:"RoomID"`
		Area        string           `json:"Area"`
		Title       string           `json:"Title"`
		Description string           `json:"Description"`
		Exits       map[string]*Exit `json:"Exits"`
		ItemIDs     []string         `json:"ItemIDs"`
	}{
		RoomID:      room.RoomID,
		Area:        room.Area,
		Title:       room.Title,
		Description: room.Description,
		Exits:       room.Exits,
		ItemIDs:     make([]string, 0, len(room.Items)),
	}

	// Collect item IDs
	for itemID := range room.Items {
		roomData.ItemIDs = append(roomData.ItemIDs, itemID)
	}

	// Serialize the room data
	serializedRoom, err := json.Marshal(roomData)
	if err != nil {
		return fmt.Errorf("error serializing room data: %w", err)
	}

	// Write the room data to the database
	err = kp.db.Update(func(tx *bolt.Tx) error {
		roomsBucket, err := tx.CreateBucketIfNotExists([]byte("Rooms"))
		if err != nil {
			return fmt.Errorf("error creating/accessing Rooms bucket: %w", err)
		}

		roomKey := strconv.FormatInt(room.RoomID, 10)
		err = roomsBucket.Put([]byte(roomKey), serializedRoom)
		if err != nil {
			return fmt.Errorf("error writing room data: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("database transaction failed: %w", err)
	}

	log.Printf("Successfully wrote room %d to database", room.RoomID)
	return nil
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

	// Display characters in the room
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

	// Display items in the room
	if len(r.Items) > 0 {
		roomInfo += "Items in the room:\n\r"
		for _, item := range r.Items {
			roomInfo += fmt.Sprintf("- %s\n\r", item.Name)
		}
	}

	return roomInfo + displayExits.String()
}

func (r *Room) RemoveItem(item *Item) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	delete(r.Items, item.ID.String())
}

func (r *Room) AddItem(item *Item) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Items[item.ID.String()] = item
}

func (s *Server) SaveActiveRooms() error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	for _, room := range s.Rooms {
		if err := s.Database.WriteRoom(room); err != nil {
			return fmt.Errorf("error saving room %d: %w", room.RoomID, err)
		}
	}

	return nil
}
