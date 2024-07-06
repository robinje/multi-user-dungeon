package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	bolt "go.etcd.io/bbolt"
)

func LoadRoomsFromJSON(fileName string) (map[int64]*Room, error) {
	byteValue, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var data struct {
		Rooms map[string]struct {
			Area        string   `json:"area"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Items       []string `json:"items"`
			Exits       []struct {
				Direction    string `json:"direction"`
				Visible      bool   `json:"visible"`
				TargetRoomID int64  `json:"target_room"`
			} `json:"exits"`
		} `json:"rooms"`
	}

	if err := json.Unmarshal(byteValue, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	rooms := make(map[int64]*Room)
	for id, roomData := range data.Rooms {
		roomID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing room ID '%s': %w", id, err)
		}
		room := &Room{
			RoomID:      roomID,
			Area:        roomData.Area,
			Title:       roomData.Title,
			Description: roomData.Description,
			Exits:       make(map[string]*Exit),
			Characters:  make(map[uint64]*Character),
			Items:       make(map[string]*Item),
		}

		for _, exitData := range roomData.Exits {
			exit := &Exit{
				TargetRoom: exitData.TargetRoomID,
				Visible:    exitData.Visible,
				Direction:  exitData.Direction,
			}
			room.Exits[exit.Direction] = exit
		}

		// Here we're just storing the item IDs. You'll need to load the actual items separately.
		for _, itemID := range roomData.Items {
			room.Items[itemID] = nil // Placeholder for the actual Item
		}

		rooms[roomID] = room
	}

	return rooms, nil
}

func (k *KeyPair) StoreRooms(rooms map[int64]*Room) error {
	return k.db.Update(func(tx *bolt.Tx) error {
		roomsBucket, err := tx.CreateBucketIfNotExists([]byte("Rooms"))
		if err != nil {
			return err
		}

		exitsBucket, err := tx.CreateBucketIfNotExists([]byte("Exits"))
		if err != nil {
			return err
		}

		for _, room := range rooms {
			roomData := struct {
				RoomID      int64    `json:"RoomID"`
				Area        string   `json:"Area"`
				Title       string   `json:"Title"`
				Description string   `json:"Description"`
				ItemIDs     []string `json:"ItemIDs"`
			}{
				RoomID:      room.RoomID,
				Area:        room.Area,
				Title:       room.Title,
				Description: room.Description,
				ItemIDs:     make([]string, 0, len(room.Items)),
			}

			for itemID := range room.Items {
				roomData.ItemIDs = append(roomData.ItemIDs, itemID)
			}

			roomBytes, err := json.Marshal(roomData)
			if err != nil {
				return fmt.Errorf("error marshaling room data: %v", err)
			}

			roomKey := strconv.FormatInt(room.RoomID, 10)
			if err := roomsBucket.Put([]byte(roomKey), roomBytes); err != nil {
				return fmt.Errorf("error writing room data: %v", err)
			}

			for dir, exit := range room.Exits {
				exitData, err := json.Marshal(exit)
				if err != nil {
					return fmt.Errorf("error marshaling exit data: %v", err)
				}

				exitKey := fmt.Sprintf("%d_%s", room.RoomID, dir)
				if err := exitsBucket.Put([]byte(exitKey), exitData); err != nil {
					return fmt.Errorf("error writing exit data: %v", err)
				}
			}
		}

		return nil
	})
}

func (kp *KeyPair) LoadRooms() (map[int64]*Room, error) {
	rooms := make(map[int64]*Room)

	err := kp.db.View(func(tx *bolt.Tx) error {
		roomsBucket := tx.Bucket([]byte("Rooms"))
		if roomsBucket == nil {
			return fmt.Errorf("rooms bucket not found")
		}

		exitsBucket := tx.Bucket([]byte("Exits"))
		if exitsBucket == nil {
			return fmt.Errorf("exits bucket not found")
		}

		return roomsBucket.ForEach(func(k, v []byte) error {
			var roomData struct {
				RoomID      int64    `json:"RoomID"`
				Area        string   `json:"Area"`
				Title       string   `json:"Title"`
				Description string   `json:"Description"`
				ItemIDs     []string `json:"ItemIDs"`
			}
			if err := json.Unmarshal(v, &roomData); err != nil {
				return fmt.Errorf("error unmarshalling room data: %w", err)
			}

			room := NewRoom(roomData.RoomID, roomData.Area, roomData.Title, roomData.Description)

			// Load items
			for _, itemID := range roomData.ItemIDs {
				item, err := kp.LoadItem(itemID, false)
				if err != nil {
					log.Printf("Warning: Failed to load item %s for room %d: %v", itemID, roomData.RoomID, err)
					continue
				}
				room.AddItem(item)
			}

			room.CleanupNilItems() // Clean up any nil items

			rooms[room.RoomID] = room
			return nil
		})
	})

	if err != nil {
		return nil, fmt.Errorf("error reading room data from BoltDB: %w", err)
	}

	return rooms, nil
}

func DisplayRooms(rooms map[int64]*Room) {
	fmt.Println("Rooms:")
	for _, room := range rooms {
		fmt.Printf("Room %d: %s\n", room.RoomID, room.Title)
		for _, exit := range room.Exits {
			fmt.Printf("  Exit %s to room %d\n", exit.Direction, exit.TargetRoom)
		}
	}
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

func SaveActiveRooms(s *Server) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	for _, room := range s.Rooms {
		if err := s.Database.WriteRoom(room); err != nil {
			return fmt.Errorf("error saving room %d: %w", room.RoomID, err)
		}
	}

	return nil
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

func Move(c *Character, direction string) {
	log.Printf("Player %s is attempting to move %s", c.Name, direction)

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if c.Room == nil {
		c.Player.ToPlayer <- "\n\rYou are not in any room to move from.\n\r"
		return
	}

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
	SendRoomMessage(oldRoom, fmt.Sprintf("\n\r%s has left going %s.\n\r", c.Name, direction))

	// Update character's room
	c.Room = newRoom

	// Safely add the character to the new room
	newRoom.Mutex.Lock()
	if newRoom.Characters == nil {
		newRoom.Characters = make(map[uint64]*Character)
	}
	newRoom.Characters[c.Index] = c
	newRoom.Mutex.Unlock()

	SendRoomMessage(newRoom, fmt.Sprintf("\n\r%s has arrived.\n\r", c.Name))

	ExecuteLookCommand(c, []string{})
}

func SendRoomMessage(r *Room, message string) {

	log.Printf("Sending message to room %d: %s", r.RoomID, message)

	for _, character := range r.Characters {
		character.Player.ToPlayer <- message

		character.Player.ToPlayer <- character.Player.Prompt

	}
}

func RoomInfo(r *Room, character *Character) string {

	if r == nil {
		log.Printf("Error: Attempted to get room info for nil room (Character: %s)", character.Name)
		return "\n\rError: You are not in a valid room.\n\r"
	}
	if character == nil {
		log.Printf("Error: Attempted to get room info for nil character (Room ID: %d)", r.RoomID)
		return "\n\rError: Invalid character.\n\r"
	}

	var roomInfo strings.Builder

	// Room Title and Description
	roomInfo.WriteString(fmt.Sprintf("\n\r[%s]\n\r%s\n\r", ApplyColor("white", r.Title), r.Description))

	// Exits
	exits := sortedExits(r)
	if len(exits) == 0 {
		roomInfo.WriteString("There are no exits.\n\r")
	} else {
		roomInfo.WriteString("Obvious exits: ")
		roomInfo.WriteString(strings.Join(exits, ", "))
		roomInfo.WriteString("\n\r")
	}

	// Characters in the room
	r.Mutex.Lock()
	otherCharacters := getOtherCharacters(r, character)
	r.Mutex.Unlock()
	if len(otherCharacters) > 0 {
		roomInfo.WriteString("Also here: ")
		roomInfo.WriteString(strings.Join(otherCharacters, ", "))
		roomInfo.WriteString("\n\r")
	} else {
		roomInfo.WriteString("You are alone.\n\r")
	}

	// Items in the room
	items := getVisibleItems(r)
	if len(items) > 0 {
		roomInfo.WriteString("Items in the room:\n\r")
		for _, item := range items {
			roomInfo.WriteString(fmt.Sprintf("- %s\n\r", item))
		}
	}

	return roomInfo.String()
}

func sortedExits(r *Room) []string {

	log.Printf("Sorting exits for room %d", r.RoomID)

	if r.Exits == nil {
		return []string{}
	}

	exits := make([]string, 0, len(r.Exits))
	for direction := range r.Exits {
		exits = append(exits, direction)
	}
	sort.Strings(exits)
	return exits
}

func getOtherCharacters(r *Room, currentCharacter *Character) []string {

	if r == nil || r.Characters == nil {
		log.Printf("Warning: Room or Characters map is nil in getOtherCharacters")
		return []string{}
	}

	otherCharacters := make([]string, 0)
	for _, c := range r.Characters {
		if c != nil && c != currentCharacter {
			otherCharacters = append(otherCharacters, c.Name)
		}
	}

	log.Printf("Found %d other characters in room %d", len(otherCharacters), r.RoomID)
	return otherCharacters
}

func getVisibleItems(r *Room) []string {
	log.Printf("Getting visible items in room %d", r.RoomID)

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if r.Items == nil {
		log.Printf("Warning: Items map is nil for room %d", r.RoomID)
		return []string{}
	}

	visibleItems := make([]string, 0, len(r.Items))
	for itemID, item := range r.Items {
		if item == nil {
			log.Printf("Warning: Nil item found with ID %s in room %d", itemID, r.RoomID)
			continue
		}
		visibleItems = append(visibleItems, item.Name)
		log.Printf("Found item %s (ID: %s) in room %d", item.Name, itemID, r.RoomID)
	}

	log.Printf("Total visible items in room %d: %d", r.RoomID, len(visibleItems))
	return visibleItems
}
