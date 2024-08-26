package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func DisplayArchetypes(archetypes *ArchetypesData) {
	for key, archetype := range archetypes.Archetypes {
		fmt.Println(key, archetype)
	}
}

func (kp *KeyPair) LoadArchetypes() (*ArchetypesData, error) {
	archetypesData := &ArchetypesData{Archetypes: make(map[string]Archetype)}

	var archetypes []Archetype
	err := kp.Scan("archetypes", &archetypes)
	if err != nil {
		return nil, fmt.Errorf("error scanning archetypes table: %w", err)
	}

	for _, archetype := range archetypes {
		archetypesData.Archetypes[archetype.Name] = archetype
		fmt.Printf("Loaded archetype '%s': %s\n", archetype.Name, archetype.Description)
	}

	return archetypesData, nil
}

func (kp *KeyPair) StoreArchetypes(archetypes *ArchetypesData) error {
	for _, archetype := range archetypes.Archetypes {
		av, err := dynamodbattribute.MarshalMap(archetype)
		if err != nil {
			return fmt.Errorf("error marshalling archetype %s: %w", archetype.Name, err)
		}

		key := map[string]*dynamodb.AttributeValue{
			"Name": {S: aws.String(archetype.Name)},
		}

		err = kp.Put("archetypes", key, av)
		if err != nil {
			return fmt.Errorf("error storing archetype %s: %w", archetype.Name, err)
		}

		log.Printf("Stored archetype: %s", archetype.Name)
	}

	return nil
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
