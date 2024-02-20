package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	bolt "go.etcd.io/bbolt"
)

type Archetype struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Attributes  map[string]float64 `json:"Attributes"`
	Abilities   map[string]float64 `json:"Abilities"`
}

type ArchetypesData struct {
	Archetypes map[string]Archetype `json:"archetypes"`
}

func loadJSONData(fileName string) (*ArchetypesData, error) {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var data ArchetypesData
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func storeArchetypesInBoltDB(dbPath string, archetypes *ArchetypesData) error {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("Archetypes"))
		if err != nil {
			return err
		}

		for key, archetype := range archetypes.Archetypes {
			data, err := json.Marshal(archetype)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(key), data); err != nil {
				return err
			}
		}
		return nil
	})
}

func main() {
	// Path to your JSON file with archetypes data
	jsonFilePath := "path_to_your_json_file.json"
	// Path to your BoltDB file
	boltDBPath := "archetypes.db"

	// Load JSON data from file
	archetypesData, err := loadJSONData(jsonFilePath)
	if err != nil {
		log.Fatalf("Failed to load JSON data: %v", err)
	}

	// Store the data in BoltDB
	err = storeArchetypesInBoltDB(boltDBPath, archetypesData)
	if err != nil {
		log.Fatalf("Failed to store data in BoltDB: %v", err)
	}

	fmt.Println("Data successfully stored in BoltDB.")
}
