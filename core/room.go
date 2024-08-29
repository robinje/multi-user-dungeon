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
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
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
			Characters:  make(map[uuid.UUID]*Character),
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

func (kp *KeyPair) StoreRooms(rooms map[int64]*Room) error {
	for _, room := range rooms {
		err := kp.WriteRoom(room)
		if err != nil {
			return fmt.Errorf("error storing room %d: %w", room.RoomID, err)
		}
	}
	return nil
}

func (kp *KeyPair) LoadRooms() (map[int64]*Room, error) {
	rooms := make(map[int64]*Room)

	var roomsData []struct {
		RoomID      int64    `dynamodbav:"RoomID"`
		Area        string   `dynamodbav:"Area"`
		Title       string   `dynamodbav:"Title"`
		Description string   `dynamodbav:"Description"`
		ItemIDs     []string `dynamodbav:"ItemIDs"`
	}

	err := kp.Scan("rooms", &roomsData)
	if err != nil {
		return nil, fmt.Errorf("error scanning rooms: %w", err)
	}

	for _, roomData := range roomsData {
		room := NewRoom(roomData.RoomID, roomData.Area, roomData.Title, roomData.Description)

		// Load exits for this room
		exits, err := kp.LoadExits(roomData.RoomID)
		if err != nil {
			Logger.Warn("Failed to load exits for room", "room_id", roomData.RoomID, "error", err)
		}
		room.Exits = exits

		// Load items
		for _, itemID := range roomData.ItemIDs {
			item, err := kp.LoadItem(itemID, false)
			if err != nil {
				Logger.Warn("Failed to load item for room", "item_id", itemID, "room_id", roomData.RoomID, "error", err)
				continue
			}
			room.AddItem(item)
		}

		room.CleanupNilItems()
		rooms[room.RoomID] = room
	}

	return rooms, nil
}

func (kp *KeyPair) LoadExits(roomID int64) (map[string]*Exit, error) {
	exits := make(map[string]*Exit)

	var exitsData []struct {
		RoomID     int64  `dynamodbav:"RoomID"`
		Direction  string `dynamodbav:"Direction"`
		TargetRoom int64  `dynamodbav:"TargetRoom"`
		Visible    bool   `dynamodbav:"Visible"`
	}

	err := kp.Query("exits", "RoomID = :roomID", map[string]*dynamodb.AttributeValue{
		":roomID": {N: aws.String(strconv.FormatInt(roomID, 10))},
	}, &exitsData)

	if err != nil {
		return nil, fmt.Errorf("error querying exits: %w", err)
	}

	for _, exitData := range exitsData {
		exits[exitData.Direction] = &Exit{
			TargetRoom: exitData.TargetRoom,
			Visible:    exitData.Visible,
			Direction:  exitData.Direction,
		}
	}

	return exits, nil
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

	for itemID := range room.Items {
		roomData.ItemIDs = append(roomData.ItemIDs, itemID)
	}

	av, err := dynamodbattribute.MarshalMap(roomData)
	if err != nil {
		return fmt.Errorf("error marshalling room data: %w", err)
	}

	key := map[string]*dynamodb.AttributeValue{
		"RoomID": {N: aws.String(strconv.FormatInt(room.RoomID, 10))},
	}

	err = kp.Put("rooms", key, av)
	if err != nil {
		return fmt.Errorf("error writing room data: %w", err)
	}

	// Write exits
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

		av, err := dynamodbattribute.MarshalMap(exitData)
		if err != nil {
			return fmt.Errorf("error marshalling exit data: %w", err)
		}

		exitKey := map[string]*dynamodb.AttributeValue{
			"RoomID":    {N: aws.String(strconv.FormatInt(room.RoomID, 10))},
			"Direction": {S: aws.String(direction)},
		}

		err = kp.Put("exits", exitKey, av)
		if err != nil {
			return fmt.Errorf("error writing exit data: %w", err)
		}
	}

	Logger.Info("Successfully wrote room to database", "room_id", room.RoomID)
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
		Characters:  make(map[uuid.UUID]*Character),
		Items:       make(map[string]*Item),
		Mutex:       sync.Mutex{},
	}

	Logger.Info("Created room", "room_title", room.Title, "room_id", room.RoomID)

	return room
}

func (r *Room) AddExit(exit *Exit) {
	r.Exits[exit.Direction] = exit
}

func Move(c *Character, direction string) {
	Logger.Info("Player is attempting to move", "player_name", c.Name, "direction", direction)

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
	delete(oldRoom.Characters, c.ID) // Changed from c.Index to c.ID
	oldRoom.Mutex.Unlock()
	SendRoomMessage(oldRoom, fmt.Sprintf("\n\r%s has left going %s.\n\r", c.Name, direction))

	// Update character's room
	c.Room = newRoom

	// Safely add the character to the new room
	newRoom.Mutex.Lock()
	if newRoom.Characters == nil {
		newRoom.Characters = make(map[uuid.UUID]*Character) // Changed from uint64 to uuid.UUID
	}
	newRoom.Characters[c.ID] = c // Changed from c.Index to c.ID
	newRoom.Mutex.Unlock()

	SendRoomMessage(newRoom, fmt.Sprintf("\n\r%s has arrived.\n\r", c.Name))

	ExecuteLookCommand(c, []string{})
}

func SendRoomMessage(r *Room, message string) {

	Logger.Info("Sending message to room", "room_id", r.RoomID, "message", message)

	for _, character := range r.Characters {
		character.Player.ToPlayer <- message

		character.Player.ToPlayer <- character.Player.Prompt

	}
}

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

func sortedExits(r *Room) []string {

	Logger.Info("Sorting exits for room", "room_id", r.RoomID)

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
