package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/dominikbraun/graph"
)

type Room struct {
	ID        int
	Area      string
	Title     string
	Narrative string
	Exits     []Exit
}

type Exit struct {
	Name         string
	TargetRoomID int
	Visible      bool
}

func main() {
	jsonFile, err := os.Open("test_data.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var data struct {
		Rooms map[string]struct {
			Area      string `json:"area"`
			Title     string `json:"title"`
			Narrative string `json:"narrative"`
			Exits     []struct {
				ExitName     string `json:"exit_name"`
				Visible      bool   `json:"visible"`
				TargetRoomID int    `json:"target_room_id"`
			} `json:"exits"`
		} `json:"rooms"`
	}
	json.Unmarshal(byteValue, &data)

	g := graph.New(graph.IntHash)
	rooms := make(map[int]*Room)

	for id, roomData := range data.Rooms {
		roomID, _ := strconv.Atoi(id)
		room := &Room{
			ID:        roomID,
			Area:      roomData.Area,
			Title:     roomData.Title,
			Narrative: roomData.Narrative,
		}
		rooms[roomID] = room
		_ = g.AddVertex(roomID)

		for _, exitData := range roomData.Exits {
			exit := Exit{
				Name:         exitData.ExitName,
				TargetRoomID: exitData.TargetRoomID,
				Visible:      exitData.Visible,
			}
			room.Exits = append(room.Exits, exit)
			_ = g.AddEdge(roomID, exit.TargetRoomID)
		}
	}

	fmt.Println("\nGraph:")
	for _, room := range rooms {
		fmt.Printf("Room %d: %s\n", room.ID, room.Title)
		for _, exit := range room.Exits {
			fmt.Printf("  Exit to %d (%s)\n", exit.TargetRoomID, exit.Name)
		}
	}
}
