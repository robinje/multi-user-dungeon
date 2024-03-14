package main

import (
	"encoding/json"
	"fmt"
	"os"

	bolt "go.etcd.io/bbolt"
)

type Object struct {
	Index       uint64
	Name        string
	Description string
	Mass        float64
	Verbs       map[string]string
	Overrides   map[string]string
	Contents    []uint64
	IsPrototype bool
}

type ObjectData struct {
	Index       uint64            `json:"index"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Mass        float64           `json:"mass"`
	Verbs       map[string]string `json:"verbs"`
	Overrides   map[string]string `json:"overrides"`
	Contents    []uint64          `json:"contents"`
	IsPrototype bool              `json:"is_prototype"`
}

type PrototypesData struct {
	ObjectPrototypes []ObjectData `json:"objectPrototypes"`
}

func protoDisplay(prototypes *PrototypesData) {
	for _, prototype := range prototypes.ObjectPrototypes {
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
	for _, prototype := range data.ObjectPrototypes {
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
		bucket, err := tx.CreateBucketIfNotExists([]byte("Prototypes"))
		if err != nil {
			return err
		}

		for _, prototype := range prototypes.ObjectPrototypes {
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
		bucket := tx.Bucket([]byte("Prototypes"))
		if bucket == nil {
			return fmt.Errorf("prototypes bucket does not exist")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var prototype ObjectData
			if err := json.Unmarshal(v, &prototype); err != nil {
				return err
			}
			fmt.Println("Reading", prototype)
			prototypesData.ObjectPrototypes = append(prototypesData.ObjectPrototypes, prototype)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return prototypesData, nil
}
