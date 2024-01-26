package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/dominikbraun/graph"
	bolt "go.etcd.io/bbolt"
)

const (
	JSON_FILE = "test_data_base.json"
	BOLT_FILE = "test_data.bolt"
)

type Index struct {
	IndexID int64
	mu      sync.Mutex
}

func (i *Index) GetID() int64 {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.IndexID++
	return i.IndexID
}

func (i *Index) SetID(id int64) {
	i.mu.Lock()
	defer i.mu.Unlock()
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

func LoadBolt(fileName string) (map[int64]*Room, []*Exit, error) {
	db, err := bolt.Open(fileName, 0600, nil)
	if err != nil {
		fmt.Printf("Error opening BoltDB file: %v\n", err)
		return nil, nil, fmt.Errorf("error opening BoltDB file: %w", err)
	}
	defer db.Close()

	rooms := make(map[int64]*Room)
	var allExits []*Exit
	g := graph.New(graph.IntHash, graph.Directed())

	err = db.View(func(tx *bolt.Tx) error {
		roomsBucket := tx.Bucket([]byte("Rooms"))
		if roomsBucket == nil {
			fmt.Println("Rooms bucket not found")
			return fmt.Errorf("Rooms bucket not found")
		}

		exitsBucket := tx.Bucket([]byte("Exits"))
		if exitsBucket == nil {
			fmt.Println("Exits bucket not found")
			return fmt.Errorf("Exits bucket not found")
		}

		err := roomsBucket.ForEach(func(k, v []byte) error {
			var room Room
			if err := json.Unmarshal(v, &room); err != nil {
				fmt.Printf("Error unmarshalling room data for key %s: %v\n", k, err)
				return fmt.Errorf("error unmarshalling room data: %w", err)
			}
			rooms[room.RoomID] = &room
			g.AddVertex(int(room.RoomID))
			// fmt.Printf("Loaded Room %d: %+v\n", room.RoomID, room)
			return nil
		})
		if err != nil {
			return err
		}

		return exitsBucket.ForEach(func(k, v []byte) error {
			var exit Exit
			if err := json.Unmarshal(v, &exit); err != nil {
				fmt.Printf("Error unmarshalling exit data for key %s: %v\n", k, err)
				return fmt.Errorf("error unmarshalling exit data: %w", err)
			}

			keyParts := strings.SplitN(string(k), "_", 2)
			if len(keyParts) != 2 {
				fmt.Printf("Invalid exit key format: %s\n", k)
				return fmt.Errorf("invalid exit key format")
			}
			roomID, err := strconv.ParseInt(keyParts[0], 10, 64)
			if err != nil {
				fmt.Printf("Error parsing room ID from key %s: %v\n", k, err)
				return fmt.Errorf("error parsing room ID from key: %w", err)
			}

			if room, exists := rooms[roomID]; exists {
				room.Exits[exit.Direction] = &exit
				g.AddEdge(int(room.RoomID), int(exit.TargetRoom), graph.EdgeData(exit.Direction))
				// fmt.Printf("Loaded Exit %s for Room %d: %+v\n", exit.Direction, room.RoomID, exit)
			} else {
				fmt.Printf("Room not found for exit key %s\n", k)
				return fmt.Errorf("room not found for exit: %s", string(k))
			}
			allExits = append(allExits, &exit)
			return nil
		})
	})

	if err != nil {
		fmt.Printf("Error reading from BoltDB: %v\n", err)
		return nil, nil, fmt.Errorf("error reading from BoltDB: %w", err)
	}

	fmt.Println("\nGraph:")
	for _, room := range rooms {
		fmt.Printf("Room %d: %s\n", room.RoomID, room.Title)
		for _, exit := range room.Exits {
			fmt.Printf("  Exit %s to room %d\n", exit.Direction, exit.TargetRoom)
		}
	}

	return rooms, allExits, nil
}

func WriteBolt(rooms map[int64]*Room, dbPath string) error {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		roomsBucket, err := tx.CreateBucketIfNotExists([]byte("Rooms"))
		if err != nil {
			fmt.Printf("Error creating 'Rooms' bucket: %v\n", err)
			return err
		}
		exitsBucket, err := tx.CreateBucketIfNotExists([]byte("Exits"))
		if err != nil {
			fmt.Printf("Error creating 'Exits' bucket: %v\n", err)
			return err
		}

		for _, room := range rooms {
			roomData, err := json.Marshal(room)
			if err != nil {
				fmt.Printf("Error marshalling room data (RoomID %d): %v\n", room.RoomID, err)
				return err
			}
			roomKey := strconv.FormatInt(room.RoomID, 10)
			// fmt.Printf("Writing Room %s: %s\n", roomKey, roomData)
			err = roomsBucket.Put([]byte(roomKey), roomData)
			if err != nil {
				fmt.Printf("Error writing room data to 'Rooms' bucket: %v\n", err)
				return err
			}

			for _, exit := range room.Exits {
				exitData, err := json.Marshal(exit)
				if err != nil {
					fmt.Printf("Error marshalling exit data (ExitID %d): %v\n", exit.ExitID, err)
					return err
				}
				exitKey := fmt.Sprintf("%d_%s", room.RoomID, exit.Direction)
				// fmt.Printf("Writing Exit %s: %s\n", exitKey, exitData)
				err = exitsBucket.Put([]byte(exitKey), exitData)
				if err != nil {
					fmt.Printf("Error writing exit data to 'Exits' bucket: %v\n", err)
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
		return // Ensure to exit if writing fails
	}
	fmt.Println("Data written successfully")

	// Load data from BoltDB
	_, _, err = LoadBolt(BOLT_FILE)
	if err != nil {
		fmt.Println("Data load from BoltDB failed:", err)
		return // Ensure to exit if loading fails
	}
	fmt.Println("Data loaded from BoltDB successfully")
}
