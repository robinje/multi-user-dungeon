package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/robinje/multi-user-dungeon/core"
)

func Move(c *core.Character, direction string) {
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
	SendRoomMessage(oldRoom, fmt.Sprintf("\n\r%s has left going %s.\n\r", c.Name, direction))

	// Update character's room
	c.Room = newRoom

	SendRoomMessage(newRoom, fmt.Sprintf("\n\r%s has arrived.\n\r", c.Name))

	// Ensure the Characters map in the new room is initialized
	newRoom.Mutex.Lock()
	if newRoom.Characters == nil {
		newRoom.Characters = make(map[uint64]*core.Character)
	}
	newRoom.Characters[c.Index] = c
	newRoom.Mutex.Unlock()

	executeLookCommand(c, []string{})
}

func SendRoomMessage(r *core.Room, message string) {

	for _, character := range r.Characters {
		character.Player.ToPlayer <- message

		character.Player.ToPlayer <- character.Player.Prompt

	}
}

func RoomInfo(r *core.Room, character *core.Character) string {
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
