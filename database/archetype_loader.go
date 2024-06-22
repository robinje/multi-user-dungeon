package main

import (
	"encoding/json"
	"fmt"
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

func archDisplay(archetypes *ArchetypesData) {
	for key, archetype := range archetypes.Archetypes {
		fmt.Println(key, archetype)
	}
}

func archLoadJSON(fileName string) (*ArchetypesData, error) {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var data ArchetypesData
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}

	// Iterate over the loaded archetypes and print a line for each.
	for key, archetype := range data.Archetypes {
		fmt.Printf("Loaded archetype '%s': %s - %s\n", key, archetype.Name, archetype.Description)
	}

	return &data, nil
}

func archWriteBolt(archetypes *ArchetypesData, dbPath string) error {
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
			fmt.Println("Writing", key, archetype)
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

func archLoadBolt(dbPath string) (*ArchetypesData, error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer db.Close()

	archetypesData := &ArchetypesData{Archetypes: make(map[string]Archetype)}

	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("Archetypes"))
		if bucket == nil {
			return fmt.Errorf("archetypes bucket does not exist")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var archetype Archetype
			if err := json.Unmarshal(v, &archetype); err != nil {
				return err
			}
			fmt.Println("Reading", string(k), archetype)
			archetypesData.Archetypes[string(k)] = archetype
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return archetypesData, nil
}
