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
	Verbs       map[string]string
	Overrides   map[string]string
}

type ObjectData struct {
	Index       uint64            `json:"index"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Mass        float64           `json:"mass"`
	Verbs       map[string]string `json:"verbs"`
	Overrides   map[string]string `json:"overrides"`
}

type Container struct {
	Index       uint64
	Name        string
	Description string
	Contents    []Object
	Mass        float64
	Verbs       map[string]string
}

type ContainerData struct {
	Index       uint64            `json:"index"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Contents    []uint64          `json:"contents"`
	Mass        float64           `json:"mass"`
	Verbs       map[string]string `json:"verbs"`
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
		Verbs:       od.Verbs,
		Overrides:   od.Overrides,
	}

	return object, nil

}

func (s *Server) WriteObject(obj *Object) error {
	// First, serialize the Object to JSON
	objData := ObjectData{
		Index:       obj.Index,
		Name:        obj.Name,
		Description: obj.Description,
		Mass:        obj.Mass,
		Verbs:       obj.Verbs,
		Overrides:   obj.Overrides,
	}
	serializedData, err := json.Marshal(objData)
	if err != nil {
		return fmt.Errorf("error marshalling object data: %v", err)
	}

	// Write serialized data to the database
	err = s.Database.db.Update(func(tx *bolt.Tx) error {
		// Ensure the "Objects" bucket exists
		bucket, err := tx.CreateBucketIfNotExists([]byte("Objects"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		// Use the object's Index as the key for the database entry
		indexKey := fmt.Sprintf("%d", obj.Index)

		// Store the serialized object data in the bucket
		err = bucket.Put([]byte(indexKey), serializedData)
		if err != nil {
			return fmt.Errorf("failed to write object data: %v", err)
		}

		return nil
	})

	return err
}

func (s *Server) LoadContainer(indexKey uint64) (*Container, error) {
	var containerData []byte

	err := s.Database.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("Containers"))
		if bucket == nil {
			return fmt.Errorf("containers bucket not found")
		}
		indexKeyStr := fmt.Sprintf("%d", indexKey)
		containerData = bucket.Get([]byte(indexKeyStr))
		return nil
	})

	if err != nil {
		return nil, err
	}

	if containerData == nil {
		return nil, fmt.Errorf("container not found")
	}

	var cd ContainerData
	if err := json.Unmarshal(containerData, &cd); err != nil {
		return nil, fmt.Errorf("error unmarshalling container data: %v", err)
	}

	container := &Container{
		Index:       cd.Index,
		Name:        cd.Name,
		Description: cd.Description,
		Mass:        cd.Mass,
		Verbs:       cd.Verbs,
	}

	// Load each object within the container
	for _, objIndex := range cd.Contents {
		obj, err := s.LoadObject(objIndex)
		if err != nil {
			return nil, fmt.Errorf("error loading container %d: %v", objIndex, err)
		}
		container.Contents = append(container.Contents, *obj)
	}

	return container, nil
}

func (s *Server) WriteContainer(container *Container) error {
	// Prepare the data for serialization
	cd := ContainerData{
		Index:       container.Index,
		Name:        container.Name,
		Description: container.Description,
		Mass:        container.Mass,
		Verbs:       container.Verbs,
	}
	for _, obj := range container.Contents {
		cd.Contents = append(cd.Contents, obj.Index)
	}

	serializedData, err := json.Marshal(cd)
	if err != nil {
		return fmt.Errorf("error marshalling container data: %v", err)
	}

	// Write serialized data to the database
	err = s.Database.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("Containers"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		indexKey := fmt.Sprintf("%d", container.Index)
		err = bucket.Put([]byte(indexKey), serializedData)
		if err != nil {
			return fmt.Errorf("failed to write container data: %v", err)
		}
		return nil
	})

	return err
}

func (s *Server) LoadObjectPrototype(indexKey uint64) (*Object, error) {
	// Load object from database

	var objectData []byte

	err := s.Database.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("ObjectPrototypes"))
		if bucket == nil {
			return fmt.Errorf("object prototypes bucket not found")
		}
		indexKey := fmt.Sprintf("%d", indexKey)
		objectData = bucket.Get([]byte(indexKey))
		return nil
	})

	if err != nil {
		return nil, err
	}

	if objectData == nil {
		return nil, fmt.Errorf("object prototype not found")
	}

	var od ObjectData
	if err := json.Unmarshal(objectData, &od); err != nil {
		return nil, fmt.Errorf("error unmarshalling object prototype data: %v", err)
	}

	object := &Object{
		Index:       od.Index,
		Name:        od.Name,
		Description: od.Description,
		Mass:        od.Mass,
		Verbs:       od.Verbs,
		Overrides:   od.Overrides,
	}

	return object, nil

}

func (s *Server) WriteObjectPrototype(obj *Object) error {
	// First, serialize the Object to JSON
	objData := ObjectData{
		Index:       obj.Index,
		Name:        obj.Name,
		Description: obj.Description,
		Mass:        obj.Mass,
		Verbs:       obj.Verbs,
		Overrides:   obj.Overrides,
	}
	serializedData, err := json.Marshal(objData)
	if err != nil {
		return fmt.Errorf("error marshalling object prototype data: %v", err)
	}

	// Write serialized data to the database
	err = s.Database.db.Update(func(tx *bolt.Tx) error {
		// Ensure the "Objects" bucket exists
		bucket, err := tx.CreateBucketIfNotExists([]byte("ObjectPrototypes"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		// Use the object's Index as the key for the database entry
		indexKey := fmt.Sprintf("%d", obj.Index)

		// Store the serialized object data in the bucket
		err = bucket.Put([]byte(indexKey), serializedData)
		if err != nil {
			return fmt.Errorf("failed to write object prototype data: %v", err)
		}

		return nil
	})

	return err
}

func (s *Server) LoadContainerPrototype(indexKey uint64) (*Container, error) {
	var containerData []byte

	err := s.Database.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("ContainerPrototypes"))
		if bucket == nil {
			return fmt.Errorf("container prototypes bucket not found")
		}
		indexKeyStr := fmt.Sprintf("%d", indexKey)
		containerData = bucket.Get([]byte(indexKeyStr))
		return nil
	})

	if err != nil {
		return nil, err
	}

	if containerData == nil {
		return nil, fmt.Errorf("container prototype not found")
	}

	var cd ContainerData
	if err := json.Unmarshal(containerData, &cd); err != nil {
		return nil, fmt.Errorf("error unmarshalling container prototype data: %v", err)
	}

	container := &Container{
		Index:       cd.Index,
		Name:        cd.Name,
		Description: cd.Description,
		Mass:        cd.Mass,
		Verbs:       cd.Verbs,
	}

	// Load each object within the container
	for _, objIndex := range cd.Contents {
		obj, err := s.LoadObject(objIndex)
		if err != nil {
			return nil, fmt.Errorf("error loading container prototype %d: %v", objIndex, err)
		}
		container.Contents = append(container.Contents, *obj)
	}

	return container, nil
}

func (s *Server) WriteContainerPrototype(container *Container) error {
	// Prepare the data for serialization
	cd := ContainerData{
		Index:       container.Index,
		Name:        container.Name,
		Description: container.Description,
		Mass:        container.Mass,
		Verbs:       container.Verbs,
	}
	for _, obj := range container.Contents {
		cd.Contents = append(cd.Contents, obj.Index)
	}

	serializedData, err := json.Marshal(cd)
	if err != nil {
		return fmt.Errorf("error marshalling container prototype data: %v", err)
	}

	// Write serialized data to the database
	err = s.Database.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("ContainerPrototypes"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		indexKey := fmt.Sprintf("%d", container.Index)
		err = bucket.Put([]byte(indexKey), serializedData)
		if err != nil {
			return fmt.Errorf("failed to write container prototype data: %v", err)
		}
		return nil
	})

	return err
}
