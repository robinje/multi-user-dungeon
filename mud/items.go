package main

import (
	"encoding/json"
	"fmt"

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
	Contents    []uint64
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
		Contents:    od.Contents,
		IsPrototype: od.IsPrototype,
		IsWorn:      od.IsWorn,
		CanPickUp:   od.CanPickUp,
	}

	return object, nil
}

func (k *KeyPair) WriteItem(obj *Item) error {
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
		Contents:    obj.Contents,
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