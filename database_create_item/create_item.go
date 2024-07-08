package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/robinje/multi-user-dungeon/core"
)

func main() {
	dbPath := "../ssh_server/data.bolt"
	kp, err := core.NewKeyPair(dbPath)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer kp.Close()

	for {
		rooms, err := kp.LoadRooms()
		if err != nil {
			fmt.Printf("Error loading rooms: %v\n", err)
			return
		}
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

		prototypes, err := kp.LoadPrototypes()
		if err != nil {
			fmt.Printf("Error loading prototypes: %v\n", err)
			return
		}
		if len(prototypes.ItemPrototypes) == 0 {
			fmt.Println("No item prototypes found. Please add some prototypes first.")
			continue
		}
		displayPrototypes(prototypes)

		prototypeID := promptForPrototype()
		if prototypeID == "" {
			continue
		}

		var selectedPrototype *core.ItemData
		for _, prototype := range prototypes.ItemPrototypes {
			if prototype.ID == prototypeID {
				selectedPrototype = &prototype
				break
			}
		}
		if selectedPrototype == nil {
			fmt.Println("Prototype not found.")
			continue
		}

		err = addItemToRoom(kp, room, selectedPrototype)
		if err != nil {
			fmt.Printf("Error adding item to room: %v\n", err)
		} else {
			fmt.Printf("Added %s to room %d.\n", selectedPrototype.Name, room.RoomID)
		}
	}
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

func displayPrototypes(prototypes *core.PrototypesData) {
	fmt.Println("Available Prototypes:")
	for _, prototype := range prototypes.ItemPrototypes {
		fmt.Printf("%s: %s\n", prototype.ID, prototype.Name)
	}
}

func promptForPrototype() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter prototype ID (empty to cancel): ")
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func addItemToRoom(kp *core.KeyPair, room *core.Room, prototype *core.ItemData) error {
	newItem := createNewItemFromPrototype(prototype)

	// Save new item to database
	err := kp.WriteItem(newItem)
	if err != nil {
		return fmt.Errorf("error saving new item: %v", err)
	}

	// Add item to room
	room.AddItem(newItem)

	// Save updated room to database
	err = kp.WriteRoom(room)
	if err != nil {
		return fmt.Errorf("error saving updated room: %v", err)
	}

	fmt.Printf("Added item %s (ID: %s) to room %d\n", newItem.Name, newItem.ID, room.RoomID)
	return nil
}

func createNewItemFromPrototype(prototype *core.ItemData) *core.Item {
	newID := uuid.New()
	newItem := &core.Item{
		ID:          newID,
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
