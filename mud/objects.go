package main

import (
	"encoding/json"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

type Object struct {
	Index       uint64
	Name        string
	Description string
	Mass        float64
}

type ObjectData struct {
	Index       uint64  `json:"index"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Mass        float64 `json:"mass"`
}

type Container struct {
	Index       uint64
	Name        string
	Description string
	Contents    []Object
	Mass        float64
}

type ContainerData struct {
	Index       uint64   `json:"index"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Contents    []uint64 `json:"contents"`
	Mass        float64  `json:"mass"`
}

func (s *Server) LoadObject(indexKey uint64) (*Object, error) {
	// Load object from database

	var objectData []byte

	err := s.Database.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("Objects"))
		if bucket == nil {
			return fmt.Errorf("objects bucket not found")
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

	var od ObjectData
	if err := json.Unmarshal(objectData, &od); err != nil {
		return nil, fmt.Errorf("error unmarshalling object data: %v", err)
	}

	object := &Object{
		Index:       od.Index,
		Name:        od.Name,
		Description: od.Description,
		Mass:        od.Mass,
	}

	return object, nil

}
