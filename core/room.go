package core

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
)

// LoadRoomsFromJSON loads rooms from a JSON file and returns a map of room IDs to Room structs.
func LoadRoomsFromJSON(fileName string) (map[int64]*Room, error) {
	// Read the JSON file
	byteValue, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Define the structure to unmarshal JSON data
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

	// Unmarshal the JSON data into the data struct
	if err := json.Unmarshal(byteValue, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	// Create a map to hold the rooms
	rooms := make(map[int64]*Room)
	for id, roomData := range data.Rooms {
		// Parse the room ID from string to int64
		roomID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing room ID '%s': %w", id, err)
		}

		// Create a new Room instance
		room := &Room{
			RoomID:      roomID,
			Area:        roomData.Area,
			Title:       roomData.Title,
			Description: roomData.Description,
			Exits:       make(map[string]*Exit),
			Characters:  make(map[uuid.UUID]*Character),
			Items:       make(map[string]*Item),
			Mutex:       sync.Mutex{},
		}

		// Add exits to the room
		for _, exitData := range roomData.Exits {
			exit := &Exit{
				TargetRoom: exitData.TargetRoomID,
				Visible:    exitData.Visible,
				Direction:  exitData.Direction,
			}
			room.Exits[exit.Direction] = exit
		}

		// Store the item IDs as placeholders; actual items will be loaded separately
		for _, itemID := range roomData.Items {
			room.Items[itemID] = nil // Placeholder for the actual Item
		}

		// Add the room to the rooms map
		rooms[roomID] = room
	}

	Logger.Info("Loaded rooms from JSON", "count", len(rooms))
	return rooms, nil
}

// StoreRooms stores all rooms into the DynamoDB database.
func (kp *KeyPair) StoreRooms(rooms map[int64]*Room) error {
	for _, room := range rooms {
		err := kp.WriteRoom(room)
		if err != nil {
			Logger.Error("Error storing room", "room_id", room.RoomID, "error", err)
			return fmt.Errorf("error storing room %d: %w", room.RoomID, err)
		}
	}
	Logger.Info("Successfully stored all rooms")
	return nil
}

// LoadRooms retrieves all rooms from the DynamoDB database.
func (kp *KeyPair) LoadRooms() (map[int64]*Room, error) {
	rooms := make(map[int64]*Room)

	var roomsData []struct {
		RoomID      int64    `dynamodbav:"RoomID"`
		Area        string   `dynamodbav:"Area"`
		Title       string   `dynamodbav:"Title"`
		Description string   `dynamodbav:"Description"`
		ItemIDs     []string `dynamodbav:"ItemIDs"`
	}

	// Scan the "rooms" table to retrieve all room data
	err := kp.Scan("rooms", &roomsData)
	if err != nil {
		Logger.Error("Error scanning rooms", "error", err)
		return nil, fmt.Errorf("error scanning rooms: %w", err)
	}

	// Process each room data entry
	for _, roomData := range roomsData {
		room := NewRoom(roomData.RoomID, roomData.Area, roomData.Title, roomData.Description)

		// Load exits for this room
		exits, err := kp.LoadExits(roomData.RoomID)
		if err != nil {
			Logger.Warn("Failed to load exits for room", "room_id", roomData.RoomID, "error", err)
		}
		room.Exits = exits

		// Load items in the room
		for _, itemID := range roomData.ItemIDs {
			item, err := kp.LoadItem(itemID, false)
			if err != nil {
				Logger.Warn("Failed to load item for room", "item_id", itemID, "room_id", roomData.RoomID, "error", err)
				continue // Skip this item and continue with others
			}
			room.AddItem(item)
		}

		// Clean up any nil items
		room.CleanupNilItems()
		rooms[room.RoomID] = room
	}

	Logger.Info("Successfully loaded rooms from database", "count", len(rooms))
	return rooms, nil
}

// LoadExits retrieves all exits for a given room from the DynamoDB database.
func (kp *KeyPair) LoadExits(roomID int64) (map[string]*Exit, error) {
	exits := make(map[string]*Exit)

	var exitsData []struct {
		RoomID     int64  `dynamodbav:"RoomID"`
		Direction  string `dynamodbav:"Direction"`
		TargetRoom int64  `dynamodbav:"TargetRoom"`
		Visible    bool   `dynamodbav:"Visible"`
	}

	// Prepare the query input
	keyCondition := "RoomID = :roomID"
	expressionAttributeValues := map[string]*dynamodb.AttributeValue{
		":roomID": {N: aws.String(strconv.FormatInt(roomID, 10))},
	}

	// Query the "exits" table for exits associated with the roomID
	err := kp.Query("exits", keyCondition, expressionAttributeValues, &exitsData)
	if err != nil {
		Logger.Error("Error querying exits", "room_id", roomID, "error", err)
		return nil, fmt.Errorf("error querying exits: %w", err)
	}

	// Process each exit data entry
	for _, exitData := range exitsData {
		exits[exitData.Direction] = &Exit{
			TargetRoom: exitData.TargetRoom,
			Visible:    exitData.Visible,
			Direction:  exitData.Direction,
		}
	}

	Logger.Info("Loaded exits for room", "room_id", roomID, "exit_count", len(exits))
	return exits, nil
}

// DisplayRooms logs information about all rooms, useful for debugging.
func DisplayRooms(rooms map[int64]*Room) {
	Logger.Info("Displaying rooms")
	for _, room := range rooms {
		Logger.Info("Room", "room_id", room.RoomID, "title", room.Title)
		for _, exit := range room.Exits {
			Logger.Info("  Exit", "direction", exit.Direction, "target_room", exit.TargetRoom)
		}
	}
}

// WriteRoom stores a single room and its exits into the DynamoDB database.
func (kp *KeyPair) WriteRoom(room *Room) error {
	// Prepare the room data with key included
	roomData := struct {
		RoomID      int64    `dynamodbav:"RoomID"`
		Area        string   `dynamodbav:"Area"`
		Title       string   `dynamodbav:"Title"`
		Description string   `dynamodbav:"Description"`
		ItemIDs     []string `dynamodbav:"ItemIDs"`
	}{
		RoomID:      room.RoomID,
		Area:        room.Area,
		Title:       room.Title,
		Description: room.Description,
		ItemIDs:     make([]string, 0, len(room.Items)),
	}

	// Collect item IDs from the room's items
	for itemID := range room.Items {
		roomData.ItemIDs = append(roomData.ItemIDs, itemID)
	}

	// Use the updated Put method which includes the key within the item
	err := kp.Put("rooms", roomData)
	if err != nil {
		Logger.Error("Error writing room data", "room_id", room.RoomID, "error", err)
		return fmt.Errorf("error writing room data: %w", err)
	}

	// Write exits associated with the room
	for direction, exit := range room.Exits {
		exitData := struct {
			RoomID     int64  `dynamodbav:"RoomID"`
			Direction  string `dynamodbav:"Direction"`
			TargetRoom int64  `dynamodbav:"TargetRoom"`
			Visible    bool   `dynamodbav:"Visible"`
		}{
			RoomID:     room.RoomID,
			Direction:  direction,
			TargetRoom: exit.TargetRoom,
			Visible:    exit.Visible,
		}

		// Use the updated Put method
		err := kp.Put("exits", exitData)
		if err != nil {
			Logger.Error("Error writing exit data", "room_id", room.RoomID, "direction", direction, "error", err)
			return fmt.Errorf("error writing exit data: %w", err)
		}
	}

	Logger.Info("Successfully wrote room to database", "room_id", room.RoomID)
	return nil
}

// SaveActiveRooms saves all active rooms to the database.
func SaveActiveRooms(s *Server) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	for _, room := range s.Rooms {
		if err := s.Database.WriteRoom(room); err != nil {
			Logger.Error("Error saving room", "room_id", room.RoomID, "error", err)
			return fmt.Errorf("error saving room %d: %w", room.RoomID, err)
		}
	}

	Logger.Info("Successfully saved all active rooms")
	return nil
}

// NewRoom creates a new Room instance with initialized fields.
func NewRoom(RoomID int64, Area string, Title string, Description string) *Room {
	room := &Room{
		RoomID:      RoomID,
		Area:        Area,
		Title:       Title,
		Description: Description,
		Exits:       make(map[string]*Exit),
		Characters:  make(map[uuid.UUID]*Character),
		Items:       make(map[string]*Item),
		Mutex:       sync.Mutex{},
	}

	Logger.Info("Created room", "room_title", room.Title, "room_id", room.RoomID)
	return room
}

// AddExit adds an exit to the room's exits map.
func (r *Room) AddExit(exit *Exit) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if exit == nil {
		Logger.Warn("Attempted to add nil exit to room", "room_id", r.RoomID)
		return
	}

	r.Exits[exit.Direction] = exit
	Logger.Info("Added exit to room", "room_id", r.RoomID, "direction", exit.Direction)
}

// Move handles character movement from one room to another based on the direction.
func Move(c *Character, direction string) {
	Logger.Info("Player is attempting to move", "player_name", c.Name, "direction", direction)

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if c.Room == nil {
		c.Player.ToPlayer <- "\n\rYou are not in any room to move from.\n\r"
		Logger.Warn("Character has no current room", "character_name", c.Name)
		return
	}

	selectedExit, exists := c.Room.Exits[direction]
	if !exists {
		c.Player.ToPlayer <- "\n\rYou cannot go that way.\n\r"
		Logger.Warn("Invalid direction for movement", "character_name", c.Name, "direction", direction)
		return
	}

	newRoom, exists := c.Server.Rooms[selectedExit.TargetRoom]
	if !exists {
		c.Player.ToPlayer <- "\n\rThe path leads nowhere.\n\r"
		Logger.Warn("Target room does not exist", "character_name", c.Name, "target_room_id", selectedExit.TargetRoom)
		return
	}

	// Safely remove the character from the old room
	oldRoom := c.Room
	oldRoom.Mutex.Lock()
	delete(oldRoom.Characters, c.ID)
	oldRoom.Mutex.Unlock()
	SendRoomMessage(oldRoom, fmt.Sprintf("\n\r%s has left going %s.\n\r", c.Name, direction))

	// Update character's room
	c.Room = newRoom

	// Safely add the character to the new room
	newRoom.Mutex.Lock()
	if newRoom.Characters == nil {
		newRoom.Characters = make(map[uuid.UUID]*Character)
	}
	newRoom.Characters[c.ID] = c
	newRoom.Mutex.Unlock()

	SendRoomMessage(newRoom, fmt.Sprintf("\n\r%s has arrived.\n\r", c.Name))

	// Let the character look around the new room
	ExecuteLookCommand(c, []string{})
	Logger.Info("Character moved successfully", "character_name", c.Name, "new_room_id", newRoom.RoomID)
}

// SendRoomMessage sends a message to all characters in the room.
func SendRoomMessage(r *Room, message string) {
	Logger.Info("Sending message to room", "room_id", r.RoomID, "message", message)

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	for _, character := range r.Characters {
		character.Player.ToPlayer <- message
		character.Player.ToPlayer <- character.Player.Prompt
	}
}

// RoomInfo generates a description of the room, including exits, characters, and items.
func RoomInfo(r *Room, character *Character) string {
	if r == nil {
		Logger.Error("Attempted to get room info for nil room", "character_name", character.Name)
		return "\n\rError: You are not in a valid room.\n\r"
	}
	if character == nil {
		Logger.Error("Attempted to get room info for nil character", "room_id", r.RoomID)
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

// sortedExits returns a sorted list of exit directions from the room.
func sortedExits(r *Room) []string {
	Logger.Info("Sorting exits for room", "room_id", r.RoomID)

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if r.Exits == nil {
		Logger.Warn("Exits map is nil for room", "room_id", r.RoomID)
		return []string{}
	}

	exits := make([]string, 0, len(r.Exits))
	for direction := range r.Exits {
		exits = append(exits, direction)
	}
	sort.Strings(exits)
	return exits
}

// getOtherCharacters returns a list of character names in the room, excluding the current character.
func getOtherCharacters(r *Room, currentCharacter *Character) []string {
	if r == nil || r.Characters == nil {
		Logger.Warn("Room or Characters map is nil in getOtherCharacters")
		return []string{}
	}

	otherCharacters := make([]string, 0)
	for _, c := range r.Characters {
		if c != nil && c != currentCharacter {
			otherCharacters = append(otherCharacters, c.Name)
		}
	}

	Logger.Info("Found other characters in room", "count", len(otherCharacters), "room_id", r.RoomID)
	return otherCharacters
}

// getVisibleItems returns a list of item names in the room.
func getVisibleItems(r *Room) []string {
	Logger.Info("Getting visible items in room", "room_id", r.RoomID)

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if r.Items == nil {
		Logger.Warn("Items map is nil for room", "room_id", r.RoomID)
		return []string{}
	}

	visibleItems := make([]string, 0, len(r.Items))
	for itemID, item := range r.Items {
		if item == nil {
			Logger.Warn("Nil item found with ID in room", "item_id", itemID, "room_id", r.RoomID)
			continue
		}
		visibleItems = append(visibleItems, item.Name)
		Logger.Info("Found item", "item_name", item.Name, "item_id", itemID)
	}

	Logger.Info("Total visible items in room", "room_id", r.RoomID, "count", len(visibleItems))
	return visibleItems
}
