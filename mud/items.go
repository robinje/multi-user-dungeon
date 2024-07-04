package main

import (
	"encoding/json"
	"fmt"
	"log"

	bolt "go.etcd.io/bbolt"
)

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

type ItemData struct {
	Index       uint64            `json:"index"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Mass        float64           `json:"mass"`
	Wearable    bool              `json:"wearable"`
	WornOn      []string          `json:"worn_on"`
	Verbs       map[string]string `json:"verbs"`
	Overrides   map[string]string `json:"overrides"`
	Container   bool              `json:"container"`
	Contents    []uint64          `json:"contents"`
	IsPrototype bool              `json:"is_prototype"`
	IsWorn      bool              `json:"is_worn"`
	CanPickUp   bool              `json:"can_pick_up"`
}

// WearLocations defines all possible locations where an item can be worn
var WearLocations = []string{
	"head",
	"neck",
	"shoulders",
	"chest",
	"back",
	"arms",
	"hands",
	"waist",
	"legs",
	"feet",
	"left_finger",
	"right_finger",
	"left_wrist",
	"right_wrist",
}

// WearLocationSet is a map for quick lookup of valid wear locations
var WearLocationSet = make(map[string]bool)

func init() {
	for _, loc := range WearLocations {
		WearLocationSet[loc] = true
	}
}

func (k *KeyPair) LoadItem(indexKey uint64, isPrototype bool) (*Item, error) {
	var objectData []byte
	bucketName := "Items"
	if isPrototype {
		bucketName = "ItemPrototypes"
	}

	err := k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("%s bucket not found", bucketName)
		}
		indexKey := fmt.Sprintf("%d", indexKey)
		objectData = bucket.Get([]byte(indexKey))
		return nil
	})

	if err != nil {
		return nil, err
	}

	if objectData == nil {
		return nil, fmt.Errorf("object not found")
	}

	var od ItemData
	if err := json.Unmarshal(objectData, &od); err != nil {
		return nil, fmt.Errorf("error unmarshalling object data: %v", err)
	}

	object := &Item{
		Index:       od.Index,
		Name:        od.Name,
		Description: od.Description,
		Mass:        od.Mass,
		Wearable:    od.Wearable,
		WornOn:      od.WornOn,
		Verbs:       od.Verbs,
		Overrides:   od.Overrides,
		Container:   od.Container,
		IsPrototype: od.IsPrototype,
		IsWorn:      od.IsWorn,
		CanPickUp:   od.CanPickUp,
	}

	// Load contents if the item is a container
	if object.Container {
		object.Contents = make([]*Item, 0, len(od.Contents))
		for _, contentIndex := range od.Contents {
			contentItem, err := k.LoadItem(contentIndex, false)
			if err != nil {
				return nil, fmt.Errorf("error loading content item %d: %v", contentIndex, err)
			}
			object.Contents = append(object.Contents, contentItem)
		}
	}

	return object, nil
}

func (k *KeyPair) WriteItem(obj *Item) error {
	contentIndices := make([]uint64, 0, len(obj.Contents))
	for _, contentItem := range obj.Contents {
		contentIndices = append(contentIndices, contentItem.Index)
		// Recursively write contained items
		if err := k.WriteItem(contentItem); err != nil {
			return fmt.Errorf("error writing content item %d: %v", contentItem.Index, err)
		}
	}

	objData := ItemData{
		Index:       obj.Index,
		Name:        obj.Name,
		Description: obj.Description,
		Mass:        obj.Mass,
		Wearable:    obj.Wearable,
		WornOn:      obj.WornOn,
		Verbs:       obj.Verbs,
		Overrides:   obj.Overrides,
		Container:   obj.Container,
		Contents:    contentIndices,
		IsPrototype: obj.IsPrototype,
		IsWorn:      obj.IsWorn,
		CanPickUp:   obj.CanPickUp,
	}

	serializedData, err := json.Marshal(objData)
	if err != nil {
		return fmt.Errorf("error marshalling object data: %v", err)
	}

	bucketName := "Items"
	if obj.IsPrototype {
		bucketName = "ItemPrototypes"
	}

	err = k.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		indexKey := fmt.Sprintf("%d", obj.Index)
		err = bucket.Put([]byte(indexKey), serializedData)
		if err != nil {
			return fmt.Errorf("failed to write object data: %v", err)
		}

		return nil
	})

	return err
}

func (s *Server) SaveActiveItems() error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	// Collect all items from rooms and characters
	itemsToSave := make(map[uint64]*Item)

	// Items in rooms
	for _, room := range s.Rooms {
		for _, item := range room.Items {
			itemsToSave[item.Index] = item
		}
	}

	// Items in character inventories
	for _, character := range s.Characters {
		for _, item := range character.Inventory {
			itemsToSave[item.Index] = item
		}
	}

	// Save all collected items
	for _, item := range itemsToSave {
		if err := s.Database.WriteItem(item); err != nil {
			return fmt.Errorf("error saving item %s (ID: %d): %w", item.Name, item.Index, err)
		}
	}

	return nil
}

func (s *Server) CreateItemFromPrototype(prototypeIndex uint64) (*Item, error) {
	prototype, err := s.Database.LoadItem(prototypeIndex, true)
	if err != nil {
		return nil, fmt.Errorf("failed to load item prototype: %w", err)
	}

	if !prototype.IsPrototype {
		return nil, fmt.Errorf("item with index %d is not a prototype", prototypeIndex)
	}

	itemIndex, err := s.Database.NextIndex("Items")
	if err != nil {
		return nil, fmt.Errorf("failed to generate item index: %w", err)
	}

	newItem := &Item{
		Index:       itemIndex,
		Name:        prototype.Name,
		Description: prototype.Description,
		Mass:        prototype.Mass,
		Wearable:    prototype.Wearable,
		WornOn:      prototype.WornOn,
		Verbs:       prototype.Verbs,
		Overrides:   prototype.Overrides,
		Container:   prototype.Container,
		IsPrototype: false,
		IsWorn:      false,
		CanPickUp:   prototype.CanPickUp,
	}

	if newItem.Container {
		newItem.Contents = make([]*Item, 0, len(prototype.Contents))
		for _, contentItem := range prototype.Contents {
			newContentItem, err := s.CreateItemFromPrototype(contentItem.Index)
			if err != nil {
				log.Printf("Error creating content item from prototype %d: %v", contentItem.Index, err)
				continue
			}
			newItem.Contents = append(newItem.Contents, newContentItem)
		}
	}

	if err := s.Database.WriteItem(newItem); err != nil {
		return nil, fmt.Errorf("failed to write new item to database: %w", err)
	}

	log.Printf("Created new item %s (ID: %d) from prototype %d", newItem.Name, newItem.Index, prototypeIndex)
	return newItem, nil
}
