package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	bolt "go.etcd.io/bbolt"
)

type Room struct {
	RoomID      int64
	Area        string
	Title       string
	Description string
	Exits       map[string]*Exit
	Items       []int64 // Added to store object IDs in the room
}

type Exit struct {
	ExitID     int64
	TargetRoom int64
	Visible    bool
	Direction  string
}

type Item struct {
	Index       uint64
	Name        string
	Description string
	Mass        float64
	Wearable    bool
	WornOn      []string
	Verbs       map[string]string
	Overrides   map[string]string
	Container   bool
	Contents    []*Item
	IsPrototype bool
	IsWorn      bool
	CanPickUp   bool
}

func main() {
	dbPath := "../mud/data.bolt"
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	// Ensure all necessary buckets exist
	err = initializeBuckets(db)
	if err != nil {
		fmt.Printf("Error initializing buckets: %v\n", err)
		return
	}

	for {
		rooms := loadRooms(db)
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

		prototypes := loadPrototypes(db)
		if len(prototypes) == 0 {
			fmt.Println("No item prototypes found. Please add some prototypes first.")
			continue
		}
		displayPrototypes(prototypes)

		prototypeID := promptForPrototype()
		if prototypeID == 0 {
			continue
		}

		prototype, exists := prototypes[prototypeID]
		if !exists {
			fmt.Println("Prototype not found.")
			continue
		}

		addItemToRoom(db, room, prototype)
		fmt.Printf("Added %s to room %d.\n", prototype.Name, room.RoomID)
	}
}

func initializeBuckets(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		buckets := []string{"Rooms", "Items", "ItemPrototypes"}
		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return fmt.Errorf("create bucket %s: %s", bucket, err)
			}
		}
		return nil
	})
}

func loadRooms(db *bolt.DB) map[int64]*Room {
	rooms := make(map[int64]*Room)
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Rooms"))
		if b == nil {
			return nil // If the bucket doesn't exist, return an empty map
		}
		b.ForEach(func(k, v []byte) error {
			var room Room
			json.Unmarshal(v, &room)
			rooms[room.RoomID] = &room
			return nil
		})
		return nil
	})
	return rooms
}

func loadPrototypes(db *bolt.DB) map[uint64]*Item {
	prototypes := make(map[uint64]*Item)
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("ItemPrototypes"))
		if b == nil {
			return nil // If the bucket doesn't exist, return an empty map
		}
		b.ForEach(func(k, v []byte) error {
			var item Item
			json.Unmarshal(v, &item)
			prototypes[item.Index] = &item
			return nil
		})
		return nil
	})
	return prototypes
}

func displayRooms(rooms map[int64]*Room) {
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

func displayPrototypes(prototypes map[uint64]*Item) {
	fmt.Println("Available Prototypes:")
	for _, prototype := range prototypes {
		fmt.Printf("%d: %s\n", prototype.Index, prototype.Name)
	}
}

func promptForPrototype() uint64 {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter prototype ID (0 to cancel): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	prototypeID, _ := strconv.ParseUint(input, 10, 64)
	return prototypeID
}

func addItemToRoom(db *bolt.DB, room *Room, prototype *Item) error {
	return db.Update(func(tx *bolt.Tx) error {
		// Create new item from prototype
		itemsBucket := tx.Bucket([]byte("Items"))
		id, _ := itemsBucket.NextSequence()
		newItem := *prototype
		newItem.Index = uint64(id)
		newItem.IsPrototype = false

		// Save new item to Items bucket
		itemData, err := json.Marshal(newItem)
		if err != nil {
			return fmt.Errorf("error marshalling new item: %v", err)
		}
		err = itemsBucket.Put([]byte(strconv.FormatUint(newItem.Index, 10)), itemData)
		if err != nil {
			return fmt.Errorf("error saving new item: %v", err)
		}

		// Update room
		room.Items = append(room.Items, int64(newItem.Index)) // Corrected line

		// Save updated room to Rooms bucket
		roomsBucket := tx.Bucket([]byte("Rooms"))
		roomData, err := json.Marshal(room)
		if err != nil {
			return fmt.Errorf("error marshalling updated room: %v", err)
		}
		err = roomsBucket.Put([]byte(strconv.FormatInt(room.RoomID, 10)), roomData)
		if err != nil {
			return fmt.Errorf("error saving updated room: %v", err)
		}

		fmt.Printf("Added item %s (ID: %d) to room %d\n", newItem.Name, newItem.Index, room.RoomID)
		return nil
	})
}
