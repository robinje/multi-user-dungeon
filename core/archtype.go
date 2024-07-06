package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	bolt "go.etcd.io/bbolt"
)

func DisplayArchetypes(archetypes *ArchetypesData) {
	for key, archetype := range archetypes.Archetypes {
		fmt.Println(key, archetype)
	}
}

func LoadArchetypesFromJSON(fileName string) (*ArchetypesData, error) {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var data ArchetypesData
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}

	for key, archetype := range data.Archetypes {
		fmt.Printf("Loaded archetype '%s': %s - %s\n", key, archetype.Name, archetype.Description)
	}

	return &data, nil
}

func (kp *KeyPair) StoreArchetypes(archetypes *ArchetypesData) error {
	kp.Mutex.Lock()
	defer kp.Mutex.Unlock()

	return kp.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("Archetypes"))
		if err != nil {
			return fmt.Errorf("create archetypes bucket: %w", err)
		}

		for key, archetype := range archetypes.Archetypes {
			data, err := json.Marshal(archetype)
			if err != nil {
				return fmt.Errorf("marshal archetype %s: %w", key, err)
			}
			if err := bucket.Put([]byte(key), data); err != nil {
				return fmt.Errorf("store archetype %s: %w", key, err)
			}
			log.Printf("Stored archetype: %s", key)
		}
		return nil
	})
}

func (kp *KeyPair) LoadArchetypes() (*ArchetypesData, error) {
	kp.Mutex.Lock()
	defer kp.Mutex.Unlock()

	archetypesData := &ArchetypesData{Archetypes: make(map[string]Archetype)}

	err := kp.db.View(func(tx *bolt.Tx) error {
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
