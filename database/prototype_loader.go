package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
)

type Item struct {
	ID          uuid.UUID
	Name        string
	Description string
	Mass        float64
	Value       uint64
	Stackable   bool
	MaxStack    uint32
	Quantity    uint32
	Wearable    bool
	WornOn      []string
	Verbs       map[string]string
	Overrides   map[string]string
	TraitMods   map[string]int8
	Container   bool
	Contents    []*Item
	IsPrototype bool
	IsWorn      bool
	CanPickUp   bool
	Metadata    map[string]string
}

type ItemData struct {
	ID          string            `json:"id"`
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
	Contents    []string          `json:"contents"`
	IsPrototype bool              `json:"is_prototype"`
	IsWorn      bool              `json:"is_worn"`
	CanPickUp   bool              `json:"can_pick_up"`
	Metadata    map[string]string `json:"metadata"`
}

type PrototypesData struct {
	ItemPrototypes []ItemData `json:"itemPrototypes"`
}

func protoDisplay(prototypes *PrototypesData) {
	for _, prototype := range prototypes.ItemPrototypes {
		fmt.Printf("ID: %s, Name: %s, Description: %s\n", prototype.ID, prototype.Name, prototype.Description)
	}
}

func protoLoadJSON(fileName string) (*PrototypesData, error) {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var data PrototypesData
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}

	// Iterate over the loaded prototypes and print a line for each.
	for _, prototype := range data.ItemPrototypes {
		fmt.Printf("Loaded prototype: %s - %s\n", prototype.Name, prototype.Description)
	}

	return &data, nil
}

func protoWriteBolt(prototypes *PrototypesData, dbPath string) error {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("ItemPrototypes"))
		if err != nil {
			return err
		}

		for _, prototype := range prototypes.ItemPrototypes {
			fmt.Println("Writing", prototype.Name)
			data, err := json.Marshal(prototype)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(prototype.ID), data); err != nil {
				return err
			}
		}
		return nil
	})
}

func protoLoadBolt(dbPath string) (*PrototypesData, error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer db.Close()

	prototypesData := &PrototypesData{}

	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("ItemPrototypes"))
		if bucket == nil {
			return fmt.Errorf("ItemPrototypes bucket does not exist")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var itemData ItemData
			if err := json.Unmarshal(v, &itemData); err != nil {
				return err
			}

			// Validate UUID
			if _, err := uuid.Parse(itemData.ID); err != nil {
				return fmt.Errorf("invalid UUID for item %s: %v", itemData.Name, err)
			}

			fmt.Printf("Reading %s (ID: %s)\n", itemData.Name, itemData.ID)
			prototypesData.ItemPrototypes = append(prototypesData.ItemPrototypes, itemData)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return prototypesData, nil
}
