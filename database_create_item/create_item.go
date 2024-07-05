package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/robinje/multi-user-dungeon/core"
)

func main() {
	dbPath := "../mud/data.bolt"
	kp, err := core.NewKeyPair(dbPath)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer kp.Close()

	err = initializeBuckets(kp)
	if err != nil {
		fmt.Printf("Error initializing buckets: %v\n", err)
		return
	}

	for {
		rooms := loadRooms(kp)
		displayRooms(rooms)

		roomID := promptForRoom()
		if roomID == 0 {
			break
		}

		room, exists := rooms[roomID]
		if !exists {
			fmt.Println("Room not found.")
			continue
		}

		prototypes := loadPrototypes(kp)
		if len(prototypes) == 0 {
			fmt.Println("No item prototypes found. Please add some prototypes first.")
			continue
		}
		displayPrototypes(prototypes)

		prototypeID := promptForPrototype()
		if prototypeID == "" {
			continue
		}

		prototype, exists := prototypes[prototypeID]
		if !exists {
			fmt.Println("Prototype not found.")
			continue
		}

		addItemToRoom(kp, room, prototype)
		fmt.Printf("Added %s to room %d.\n", prototype.Name, room.RoomID)
	}
}

func initializeBuckets(kp *core.KeyPair) error {
	buckets := []string{"Rooms", "Items", "ItemPrototypes"}
	for _, bucket := range buckets {
		err := kp.Put(bucket, []byte("init"), []byte("init"))
		if err != nil {
			return fmt.Errorf("create bucket %s: %s", bucket, err)
		}
		kp.Delete(bucket, []byte("init"))
	}
	return nil
}

func loadRooms(kp *core.KeyPair) map[int64]*core.Room {
	rooms := make(map[int64]*core.Room)
	roomsData, err := kp.Get("Rooms", nil)
	if err != nil {
		return rooms
	}
	json.Unmarshal(roomsData, &rooms)
	return rooms
}

func loadPrototypes(kp *core.KeyPair) map[string]*core.Item {
	prototypes := make(map[string]*core.Item)
	prototypesData, err := kp.Get("ItemPrototypes", nil)
	if err != nil {
		return prototypes
	}
	json.Unmarshal(prototypesData, &prototypes)
	return prototypes
}

func displayRooms(rooms map[int64]*core.Room) {
	fmt.Println("Available Rooms:")
	for _, room := range rooms {
		fmt.Printf("%d: %s\n", room.RoomID, room.Title)
	}
}

func promptForRoom() int64 {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter room ID (0 to quit): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	roomID, _ := strconv.ParseInt(input, 10, 64)
	return roomID
}

func displayPrototypes(prototypes map[string]*core.Item) {
	fmt.Println("Available Prototypes:")
	for id, prototype := range prototypes {
		fmt.Printf("%s: %s\n", id, prototype.Name)
	}
}

func promptForPrototype() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter prototype ID (empty to cancel): ")
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func addItemToRoom(kp *core.KeyPair, room *core.Room, prototype *core.Item) error {
	newItem := createNewItemFromPrototype(prototype)

	// Save new item to Items bucket
	itemData, err := json.Marshal(newItem)
	if err != nil {
		return fmt.Errorf("error marshalling new item: %v", err)
	}
	err = kp.Put("Items", []byte(newItem.ID.String()), itemData)
	if err != nil {
		return fmt.Errorf("error saving new item: %v", err)
	}

	// Update room
	if room.Items == nil {
		room.Items = make(map[string]*core.Item)
	}
	room.Items[newItem.ID.String()] = newItem

	// Save updated room to Rooms bucket
	roomData, err := json.Marshal(room)
	if err != nil {
		return fmt.Errorf("error marshalling updated room: %v", err)
	}
	err = kp.Put("Rooms", []byte(strconv.FormatInt(room.RoomID, 10)), roomData)
	if err != nil {
		return fmt.Errorf("error saving updated room: %v", err)
	}

	fmt.Printf("Added item %s (ID: %s) to room %d\n", newItem.Name, newItem.ID, room.RoomID)
	return nil
}

func createNewItemFromPrototype(prototype *core.Item) *core.Item {
	newItem := &core.Item{
		ID:          uuid.New(),
		Name:        prototype.Name,
		Description: prototype.Description,
		Mass:        prototype.Mass,
		Value:       prototype.Value,
		Stackable:   prototype.Stackable,
		MaxStack:    prototype.MaxStack,
		Quantity:    1,
		Wearable:    prototype.Wearable,
		WornOn:      make([]string, len(prototype.WornOn)),
		Verbs:       make(map[string]string),
		Overrides:   make(map[string]string),
		TraitMods:   make(map[string]int8),
		Container:   prototype.Container,
		IsPrototype: false,
		IsWorn:      false,
		CanPickUp:   prototype.CanPickUp,
		Contents:    make([]*core.Item, 0),
		Metadata:    make(map[string]string),
	}

	// Deep copy relevant fields
	copy(newItem.WornOn, prototype.WornOn)
	for k, v := range prototype.Verbs {
		newItem.Verbs[k] = v
	}
	for k, v := range prototype.Overrides {
		newItem.Overrides[k] = v
	}
	for k, v := range prototype.TraitMods {
		newItem.TraitMods[k] = v
	}
	for k, v := range prototype.Metadata {
		newItem.Metadata[k] = v
	}

	return newItem
}
