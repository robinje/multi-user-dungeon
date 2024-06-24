package main

import (
	"encoding/json"
	"fmt"
	"os"

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

type PrototypesData struct {
	ItemPrototypes []ItemData `json:"itemPrototypes"`
}

func protoDisplay(prototypes *PrototypesData) {
	for _, prototype := range prototypes.ItemPrototypes {
		fmt.Println(prototype)
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
			fmt.Println("Writing", prototype)
			data, err := json.Marshal(prototype)
			if err != nil {
				return err
			}
			key := []byte(fmt.Sprintf("%d", prototype.Index))
			if err := bucket.Put(key, data); err != nil {
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

			item := &Item{
				Index:       itemData.Index,
				Name:        itemData.Name,
				Description: itemData.Description,
				Mass:        itemData.Mass,
				Wearable:    itemData.Wearable,
				WornOn:      itemData.WornOn,
				Verbs:       itemData.Verbs,
				Overrides:   itemData.Overrides,
				Container:   itemData.Container,
				IsPrototype: itemData.IsPrototype,
				IsWorn:      itemData.IsWorn,
				CanPickUp:   itemData.CanPickUp,
			}

			if item.Container {
				item.Contents = make([]*Item, 0, len(itemData.Contents))
				for _, contentIndex := range itemData.Contents {
					// Note: This is a simplified version. You might need to implement
					// a recursive loading mechanism for nested items.
					contentItem := &Item{Index: contentIndex}
					item.Contents = append(item.Contents, contentItem)
				}
			}

			fmt.Println("Reading", item)
			prototypesData.ItemPrototypes = append(prototypesData.ItemPrototypes, itemData)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return prototypesData, nil
}
