package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
)

type Room struct {
	RoomID      int64
	Area        string
	Title       string
	Description string
	Exits       map[string]*Exit
	ItemIDs     []string // Changed to store UUIDs as strings
}

type Exit struct {
	ExitID     int64
	TargetRoom int64
	Visible    bool
	Direction  string
}

type Item struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Mass        float64           `json:"mass"`
	Value       uint64            `json:"value"`
	Stackable   bool              `json:"stackable"`
	MaxStack    uint32            `json:"max_stack"`
	Quantity    uint32            `json:"quantity"`
	Wearable    bool              `json:"wearable"`
	WornOn      []string          `json:"worn_on"`
	Verbs       map[string]string `json:"verbs"`
	Overrides   map[string]string `json:"overrides"`
	TraitMods   map[string]int8   `json:"trait_mods"`
	Container   bool              `json:"container"`
	Contents    []string          `json:"contents,omitempty"`
	IsPrototype bool              `json:"is_prototype"`
	IsWorn      bool              `json:"is_worn"`
	CanPickUp   bool              `json:"can_pick_up"`
	Metadata    map[string]string `json:"metadata"`
}

func main() {
	dbPath := "../mud/data.bolt"
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

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
		if prototypeID == "" {
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

func loadPrototypes(db *bolt.DB) map[string]*Item {
	prototypes := make(map[string]*Item)
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("ItemPrototypes"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var item Item
			if err := json.Unmarshal(v, &item); err != nil {
				return err
			}
			prototypes[item.ID.String()] = &item
			return nil
		})
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

func displayPrototypes(prototypes map[string]*Item) {
	fmt.Println("Available Prototypes:")
	for id, prototype := range prototypes {
		fmt.Printf("%s: %s\n", id, prototype.Name)
	}
}

// TODO: Index the prototypes by an Integer value for easier selection
func promptForPrototype() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter prototype ID (empty to cancel): ")
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func addItemToRoom(db *bolt.DB, room *Room, prototype *Item) error {
	return db.Update(func(tx *bolt.Tx) error {
		itemsBucket, err := tx.CreateBucketIfNotExists([]byte("Items"))
		if err != nil {
			return fmt.Errorf("failed to create items bucket: %v", err)
		}

		newItem := &Item{
			ID:          uuid.New(),
			Name:        prototype.Name,
			Description: prototype.Description,
			Mass:        prototype.Mass,
			Value:       prototype.Value,
			Stackable:   prototype.Stackable,
			MaxStack:    prototype.MaxStack,
			Quantity:    1, // Start with quantity 1 for new items
			Wearable:    prototype.Wearable,
			WornOn:      make([]string, len(prototype.WornOn)),
			Verbs:       make(map[string]string),
			Overrides:   make(map[string]string),
			TraitMods:   make(map[string]int8),
			Container:   prototype.Container,
			IsPrototype: false,
			IsWorn:      false,
			CanPickUp:   prototype.CanPickUp,
			Contents:    make([]string, 0),
			Metadata:    make(map[string]string),
		}

		// Deep copy slices and maps
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

		if newItem.Container && len(prototype.Contents) > 0 {
			// Handle contents for container items
			for _, contentID := range prototype.Contents {
				// Load the content item prototype
				contentPrototype, err := loadItemPrototype(tx, contentID)
				if err != nil {
					return fmt.Errorf("error loading content item prototype: %v", err)
				}

				// Create a new item based on the content prototype
				newContentItem, err := createNewItemFromPrototype(tx, contentPrototype)
				if err != nil {
					return fmt.Errorf("error creating new content item: %v", err)
				}

				// Add the new content item's ID to the container's contents
				newItem.Contents = append(newItem.Contents, newContentItem.ID.String())
			}
		}

		// Save new item to Items bucket
		itemData, err := json.Marshal(newItem)
		if err != nil {
			return fmt.Errorf("error marshalling new item: %v", err)
		}
		err = itemsBucket.Put([]byte(newItem.ID.String()), itemData)
		if err != nil {
			return fmt.Errorf("error saving new item: %v", err)
		}

		// Update room
		room.ItemIDs = append(room.ItemIDs, newItem.ID.String())

		// Save updated room to Rooms bucket
		roomsBucket := tx.Bucket([]byte("Rooms"))
		if roomsBucket == nil {
			return fmt.Errorf("rooms bucket not found")
		}
		roomData, err := json.Marshal(room)
		if err != nil {
			return fmt.Errorf("error marshalling updated room: %v", err)
		}
		err = roomsBucket.Put([]byte(strconv.FormatInt(room.RoomID, 10)), roomData)
		if err != nil {
			return fmt.Errorf("error saving updated room: %v", err)
		}

		fmt.Printf("Added item %s (ID: %s) to room %d\n", newItem.Name, newItem.ID, room.RoomID)
		return nil
	})
}

func loadItemPrototype(tx *bolt.Tx, id string) (*Item, error) {
	prototypesBucket := tx.Bucket([]byte("ItemPrototypes"))
	if prototypesBucket == nil {
		return nil, fmt.Errorf("item prototypes bucket not found")
	}

	prototypeData := prototypesBucket.Get([]byte(id))
	if prototypeData == nil {
		return nil, fmt.Errorf("item prototype with ID %s not found", id)
	}

	var prototype Item
	if err := json.Unmarshal(prototypeData, &prototype); err != nil {
		return nil, fmt.Errorf("error unmarshalling item prototype: %v", err)
	}

	return &prototype, nil
}

func createNewItemFromPrototype(tx *bolt.Tx, prototype *Item) (*Item, error) {
	newItem := &Item{
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
		Contents:    make([]string, 0),
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

	// Save the new item
	itemsBucket, err := tx.CreateBucketIfNotExists([]byte("Items"))
	if err != nil {
		return nil, fmt.Errorf("failed to create or access items bucket: %v", err)
	}

	itemData, err := json.Marshal(newItem)
	if err != nil {
		return nil, fmt.Errorf("error marshalling new item: %v", err)
	}

	err = itemsBucket.Put([]byte(newItem.ID.String()), itemData)
	if err != nil {
		return nil, fmt.Errorf("error saving new item: %v", err)
	}

	return newItem, nil
}
