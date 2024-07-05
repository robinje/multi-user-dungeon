package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	bolt "go.etcd.io/bbolt"
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
			Characters:  make(map[uint64]*Character),
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

func (k *KeyPair) StoreRooms(rooms map[int64]*Room) error {
	return k.db.Update(func(tx *bolt.Tx) error {
		roomsBucket, err := tx.CreateBucketIfNotExists([]byte("Rooms"))
		if err != nil {
			return err
		}

		exitsBucket, err := tx.CreateBucketIfNotExists([]byte("Exits"))
		if err != nil {
			return err
		}

		for _, room := range rooms {
			roomData := struct {
				RoomID      int64    `json:"RoomID"`
				Area        string   `json:"Area"`
				Title       string   `json:"Title"`
				Description string   `json:"Description"`
				ItemIDs     []string `json:"ItemIDs"`
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

			roomBytes, err := json.Marshal(roomData)
			if err != nil {
				return fmt.Errorf("error marshaling room data: %v", err)
			}

			roomKey := strconv.FormatInt(room.RoomID, 10)
			if err := roomsBucket.Put([]byte(roomKey), roomBytes); err != nil {
				return fmt.Errorf("error writing room data: %v", err)
			}

			for dir, exit := range room.Exits {
				exitData, err := json.Marshal(exit)
				if err != nil {
					return fmt.Errorf("error marshaling exit data: %v", err)
				}

				exitKey := fmt.Sprintf("%d_%s", room.RoomID, dir)
				if err := exitsBucket.Put([]byte(exitKey), exitData); err != nil {
					return fmt.Errorf("error writing exit data: %v", err)
				}
			}
		}

		return nil
	})
}

func (k *KeyPair) LoadRooms() (map[int64]*Room, error) {
	rooms := make(map[int64]*Room)

	err := k.db.View(func(tx *bolt.Tx) error {
		roomsBucket := tx.Bucket([]byte("Rooms"))
		if roomsBucket == nil {
			return fmt.Errorf("rooms bucket not found")
		}

		exitsBucket := tx.Bucket([]byte("Exits"))
		if exitsBucket == nil {
			return fmt.Errorf("exits bucket not found")
		}

		err := roomsBucket.ForEach(func(k, v []byte) error {
			var roomData struct {
				RoomID      int64    `json:"RoomID"`
				Area        string   `json:"Area"`
				Title       string   `json:"Title"`
				Description string   `json:"Description"`
				ItemIDs     []string `json:"ItemIDs"`
			}
			if err := json.Unmarshal(v, &roomData); err != nil {
				return fmt.Errorf("error unmarshalling room data: %w", err)
			}

			room := &Room{
				RoomID:      roomData.RoomID,
				Area:        roomData.Area,
				Title:       roomData.Title,
				Description: roomData.Description,
				Exits:       make(map[string]*Exit),
				Characters:  make(map[uint64]*Character),
				Items:       make(map[string]*Item),
			}

			for _, itemID := range roomData.ItemIDs {
				room.Items[itemID] = nil // Placeholder for the actual Item
			}

			rooms[room.RoomID] = room
			return nil
		})
		if err != nil {
			return err
		}

		return exitsBucket.ForEach(func(k, v []byte) error {
			var exit Exit
			if err := json.Unmarshal(v, &exit); err != nil {
				return fmt.Errorf("error unmarshalling exit data: %w", err)
			}

			keyParts := strings.SplitN(string(k), "_", 2)
			if len(keyParts) != 2 {
				return fmt.Errorf("invalid exit key format")
			}
			roomID, err := strconv.ParseInt(keyParts[0], 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing room ID from key: %w", err)
			}

			if room, exists := rooms[roomID]; exists {
				room.Exits[exit.Direction] = &exit
			} else {
				return fmt.Errorf("room not found for exit: %s", string(k))
			}
			return nil
		})
	})

	if err != nil {
		return nil, fmt.Errorf("error reading room data from BoltDB: %w", err)
	}

	return rooms, nil
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
