package core

import (
	"encoding/json"
	"fmt"
	"os"
)

// DisplayArchetypes logs the loaded archetypes for debugging purposes.
func DisplayArchetypes(archetypes *ArchetypesData) {
	for key, archetype := range archetypes.Archetypes {
		Logger.Debug("Archetype", "name", key, "description", archetype.Description)
	}
}

// LoadArchetypes retrieves all archetypes from the DynamoDB table and returns them as an ArchetypesData struct.
func (kp *KeyPair) LoadArchetypes() (*ArchetypesData, error) {
	archetypesData := &ArchetypesData{Archetypes: make(map[string]Archetype)}

	var archetypes []Archetype
	err := kp.Scan("archetypes", &archetypes)
	if err != nil {
		return nil, fmt.Errorf("error scanning archetypes table: %w", err)
	}

	for _, archetype := range archetypes {
		archetypesData.Archetypes[archetype.ArchetypeName] = archetype
		Logger.Debug("Loaded archetype", "name", archetype.ArchetypeName, "description", archetype.Description)
	}

	return archetypesData, nil
}

// StoreArchetypes stores all archetypes into the DynamoDB table.
func (kp *KeyPair) StoreArchetypes(archetypes *ArchetypesData) error {
	for _, archetype := range archetypes.Archetypes {
		err := kp.Put("archetypes", archetype)
		if err != nil {
			return fmt.Errorf("error storing archetype %s: %w", archetype.ArchetypeName, err)
		}

		Logger.Info("Stored archetype", "name", archetype.ArchetypeName)
	}

	return nil
}

// LoadArchetypesFromJSON loads archetypes from a JSON file and returns them as an ArchetypesData struct.
func LoadArchetypesFromJSON(fileName string) (*ArchetypesData, error) {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("error reading JSON file %s: %w", fileName, err)
	}

	var data ArchetypesData
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON data: %w", err)
	}

	for key, archetype := range data.Archetypes {
		Logger.Debug("Loaded archetype from JSON", "name", key, "description", archetype.Description)
	}

	return &data, nil
}
