package core

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
)

func DisplayPrototypes(prototypes *PrototypesData) {
	for _, prototype := range prototypes.ItemPrototypes {
		fmt.Printf("ID: %s, Name: %s, Description: %s\n", prototype.ID, prototype.Name, prototype.Description)
	}
}

func LoadPrototypesFromJSON(fileName string) (*PrototypesData, error) {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var data PrototypesData
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}

	for _, prototype := range data.ItemPrototypes {
		fmt.Printf("Loaded prototype: %s - %s\n", prototype.Name, prototype.Description)
	}

	return &data, nil
}

func StorePrototypes(kp *KeyPair, prototypes *PrototypesData) error {
	return kp.db.Update(func(tx *bolt.Tx) error {
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

func LoadPrototypes(kp *KeyPair) (*PrototypesData, error) {
	prototypesData := &PrototypesData{}

	err := kp.db.View(func(tx *bolt.Tx) error {
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

func (k *KeyPair) StorePrototypes(prototypes *PrototypesData) error {
	return StorePrototypes(k, prototypes)
}

func (k *KeyPair) LoadPrototypes() (*PrototypesData, error) {
	return LoadPrototypes(k)
}
