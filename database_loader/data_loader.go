package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/dominikbraun/graph"
	bolt "go.etcd.io/bbolt"
)

const (
	JSON_FILE = "test_data.json"
	BOLT_FILE = "test_data.bolt"
)

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

func LoadJSON(fileName string) (map[int64]*Room, []*Exit, error) {

	// Read the entire file
	byteValue, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return nil, nil, err
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
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return nil, nil, err
	}

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

	var allExits []*Exit
	for _, room := range rooms {
		for _, exit := range room.Exits {
			allExits = append(allExits, exit)
		}
	}

	fmt.Println("\nGraph:")
	for _, room := range rooms {
		fmt.Printf("Room %d: %s\n", room.RoomID, room.Title)
		for _, exit := range room.Exits {
			fmt.Printf("  Exit %s to room %d (%s)\n", exit.Direction, exit.TargetRoom, rooms[exit.TargetRoom].Title)
		}
	}

	return rooms, allExits, nil
}

func WriteBolt(rooms map[int64]*Room, dbPath string) error {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		roomsBucket, err := tx.CreateBucketIfNotExists([]byte("Rooms"))
		if err != nil {
			return err
		}
		exitsBucket, err := tx.CreateBucketIfNotExists([]byte("Exits"))
		if err != nil {
			return err
		}

		for _, room := range rooms {
			roomData, err := json.Marshal(room)
			if err != nil {
				return err
			}
			err = roomsBucket.Put([]byte(strconv.FormatInt(room.RoomID, 10)), roomData)
			if err != nil {
				return err
			}

			for _, exit := range room.Exits {
				exitData, err := json.Marshal(exit)
				if err != nil {
					return err
				}
				key := fmt.Sprintf("%d_%d", room.RoomID, exit.ExitID)
				err = exitsBucket.Put([]byte(key), exitData)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func main() {
	// Load the JSON data
	rooms, _, err := LoadJSON(JSON_FILE)
	if err != nil {
		fmt.Println("Data load failed:", err)
		return // Ensure to exit if loading fails
	}
	fmt.Println("Data loaded successfully")

	// Write data to BoltDB
	if err := WriteBolt(rooms, BOLT_FILE); err != nil {
		fmt.Println("Data write failed:", err)
	} else {
		fmt.Println("Data written successfully")
	}
}
