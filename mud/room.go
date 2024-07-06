package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/robinje/multi-user-dungeon/core"
)

func Move(c *core.Character, direction string) {

	log.Printf("Player %s is attempting to move %s", c.Name, direction)

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

	log.Printf("Sending message to room %d: %s", r.RoomID, message)

	for _, character := range r.Characters {
		character.Player.ToPlayer <- message

		character.Player.ToPlayer <- character.Player.Prompt

	}
}

func RoomInfo(r *core.Room, character *core.Character) string {

	log.Printf("Generating room info for character %s in room %d", character.Name, r.RoomID)

	if r == nil {
		log.Printf("Error: Attempted to get room info for nil room (Character: %s)", character.Name)
		return "\n\rError: You are not in a valid room.\n\r"
	}
	if character == nil {
		log.Printf("Error: Attempted to get room info for nil character (Room ID: %d)", r.RoomID)
		return "\n\rError: Invalid character.\n\r"
	}

	log.Printf("Generating room info for character %s in room %d", character.Name, r.RoomID)

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
	otherCharacters := getOtherCharacters(r, character)
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

func sortedExits(r *core.Room) []string {

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

func getOtherCharacters(r *core.Room, currentCharacter *core.Character) []string {

	log.Printf("Getting other characters in room %d", r.RoomID)

	if r == nil || r.Characters == nil {
		log.Printf("Warning: Room or Characters map is nil in getOtherCharacters")
		return []string{}
	}

	log.Printf("Room: %v", r)

	otherCharacters := make([]string, 0)
	for _, c := range r.Characters {
		if c != nil && c != currentCharacter {
			otherCharacters = append(otherCharacters, c.Name)
		}
	}

	log.Printf("Found %d other characters in room %d", len(otherCharacters), r.RoomID)
	return otherCharacters
}

func getVisibleItems(r *core.Room) []string {

	log.Printf("Getting visible items in room %d", r.RoomID)

	if r.Items == nil {
		return []string{}
	}

	visibleItems := make([]string, 0, len(r.Items))
	for _, item := range r.Items {
		if item != nil {
			visibleItems = append(visibleItems, item.Name)
		}
	}
	return visibleItems
}
