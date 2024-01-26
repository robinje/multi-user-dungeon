package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/dominikbraun/graph"
)

const DATA_FILE = "test_data.json"

type Index struct {
	IndexID int64
}

func (i *Index) GetID() int64 {
	i.IndexID++
	return i.IndexID
}

func (i *Index) SetID(id int64) {
	if id > i.IndexID {
		i.IndexID = id
	}
}

type Exit struct {
	ExitID     int64
	TargetRoom int64
	Visible    bool
	Direction  string
}
type Room struct {
	RoomID      int64
	Area        string
	Title       string
	Description string
	Exits       map[string]*Exit
}

func main() {
	// Read the entire file
	byteValue, err := os.ReadFile(DATA_FILE)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	index := &Index{
		IndexID: 0,
	}

	var data struct {
		Rooms map[string]struct {
			Area      string `json:"area"`
			Title     string `json:"title"`
			Narrative string `json:"description"`
			Exits     []struct {
				ExitName     string `json:"direction"`
				Visible      bool   `json:"visible"`
				TargetRoomID int64  `json:"target_room"`
			} `json:"exits"`
		} `json:"rooms"`
	}
	json.Unmarshal(byteValue, &data)

	g := graph.New(graph.IntHash, graph.Directed())
	rooms := make(map[int64]*Room)

	index.SetID(int64(len(data.Rooms)))

	for id, roomData := range data.Rooms {
		roomID, _ := strconv.Atoi(id)
		room := &Room{
			RoomID:      int64(roomID),
			Area:        roomData.Area,
			Title:       roomData.Title,
			Description: roomData.Narrative,
			Exits:       make(map[string]*Exit),
		}
		rooms[int64(roomID)] = room
		_ = g.AddVertex(int(roomID))

		for _, exitData := range roomData.Exits {
			exit := Exit{
				ExitID:     index.GetID(),
				TargetRoom: exitData.TargetRoomID,
				Visible:    exitData.Visible,
				Direction:  exitData.ExitName,
			}
			room.Exits[exit.Direction] = &exit
			_ = g.AddEdge(roomID, int(exit.TargetRoom), graph.EdgeData(exit.Direction))
		}
	}

	fmt.Println("\nGraph:")
	for _, room := range rooms {
		fmt.Printf("Room %d: %s\n", room.RoomID, room.Title)
		for _, exit := range room.Exits {
			fmt.Printf("  Exit %s to room %d (%s)\n", exit.Direction, exit.TargetRoom, rooms[exit.TargetRoom].Title)
		}
	}
}
