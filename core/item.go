package core

import (
	"encoding/json"
	"fmt"
	"log"
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

func (kp *KeyPair) StorePrototypes(prototypes *PrototypesData) error {
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

func (kp *KeyPair) LoadPrototypes() (*PrototypesData, error) {
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

func (k *KeyPair) LoadItem(id string, isPrototype bool) (*Item, error) {
	var objectData []byte
	bucketName := "Items"
	if isPrototype {
		bucketName = "ItemPrototypes"
	}

	err := k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("%s bucket not found", bucketName)
		}
		objectData = bucket.Get([]byte(id))
		return nil
	})

	if err != nil {
		return nil, err
	}

	if objectData == nil {
		return nil, fmt.Errorf("object not found")
	}

	var od ItemData
	if err := json.Unmarshal(objectData, &od); err != nil {
		return nil, fmt.Errorf("error unmarshalling object data: %v", err)
	}

	itemID, err := uuid.Parse(od.ID)
	if err != nil {
		return nil, fmt.Errorf("error parsing item ID: %v", err)
	}

	object := &Item{
		ID:          itemID,
		Name:        od.Name,
		Description: od.Description,
		Mass:        od.Mass,
		Value:       od.Value,
		Stackable:   od.Stackable,
		MaxStack:    od.MaxStack,
		Quantity:    od.Quantity,
		Wearable:    od.Wearable,
		WornOn:      od.WornOn,
		Verbs:       od.Verbs,
		Overrides:   od.Overrides,
		TraitMods:   od.TraitMods,
		Container:   od.Container,
		IsPrototype: od.IsPrototype,
		IsWorn:      od.IsWorn,
		CanPickUp:   od.CanPickUp,
		Metadata:    od.Metadata,
	}

	// Load contents if the item is a container
	if object.Container {
		object.Contents = make([]*Item, 0, len(od.Contents))
		for _, contentID := range od.Contents {
			contentItem, err := k.LoadItem(contentID, false)
			if err != nil {
				return nil, fmt.Errorf("error loading content item %s: %v", contentID, err)
			}
			object.Contents = append(object.Contents, contentItem)
		}
	}

	return object, nil
}

func (k *KeyPair) WriteItem(obj *Item) error {
	contentIDs := make([]string, 0, len(obj.Contents))
	for _, contentItem := range obj.Contents {
		contentIDs = append(contentIDs, contentItem.ID.String())
		// Recursively write contained items
		if err := k.WriteItem(contentItem); err != nil {
			return fmt.Errorf("error writing content item %s: %v", contentItem.ID, err)
		}
	}

	objData := ItemData{
		ID:          obj.ID.String(),
		Name:        obj.Name,
		Description: obj.Description,
		Mass:        obj.Mass,
		Value:       obj.Value,
		Stackable:   obj.Stackable,
		MaxStack:    obj.MaxStack,
		Quantity:    obj.Quantity,
		Wearable:    obj.Wearable,
		WornOn:      obj.WornOn,
		Verbs:       obj.Verbs,
		Overrides:   obj.Overrides,
		TraitMods:   obj.TraitMods,
		Container:   obj.Container,
		Contents:    contentIDs,
		IsPrototype: obj.IsPrototype,
		IsWorn:      obj.IsWorn,
		CanPickUp:   obj.CanPickUp,
		Metadata:    obj.Metadata,
	}

	serializedData, err := json.Marshal(objData)
	if err != nil {
		return fmt.Errorf("error marshalling object data: %v", err)
	}

	bucketName := "Items"
	if obj.IsPrototype {
		bucketName = "ItemPrototypes"
	}

	err = k.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		err = bucket.Put([]byte(obj.ID.String()), serializedData)
		if err != nil {
			return fmt.Errorf("failed to write object data: %v", err)
		}

		return nil
	})

	return err
}

func (s *Server) SaveActiveItems() error {
	// Collect all items from rooms and characters
	itemsToSave := make(map[uuid.UUID]*Item)

	// Items in rooms
	for _, room := range s.Rooms {
		for _, item := range room.Items {
			itemsToSave[item.ID] = item
		}
	}

	// Items in character inventories
	for _, character := range s.Characters {
		for _, item := range character.Inventory {
			itemsToSave[item.ID] = item
		}
	}

	// Save all collected items
	for _, item := range itemsToSave {
		if err := s.Database.WriteItem(item); err != nil {
			return fmt.Errorf("error saving item %s (ID: %s): %w", item.Name, item.ID, err)
		}
	}

	return nil
}

func (s *Server) CreateItemFromPrototype(prototypeID string) (*Item, error) {
	prototype, err := s.Database.LoadItem(prototypeID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to load item prototype: %w", err)
	}

	if !prototype.IsPrototype {
		return nil, fmt.Errorf("item with ID %s is not a prototype", prototypeID)
	}

	newItem := &Item{
		ID:          uuid.New(),
		Name:        prototype.Name,
		Description: prototype.Description,
		Mass:        prototype.Mass,
		Value:       prototype.Value,
		Stackable:   prototype.Stackable,
		MaxStack:    prototype.MaxStack,
		Quantity:    prototype.Quantity,
		Wearable:    prototype.Wearable,
		WornOn:      prototype.WornOn,
		Verbs:       prototype.Verbs,
		Overrides:   prototype.Overrides,
		TraitMods:   make(map[string]int8),
		Container:   prototype.Container,
		IsPrototype: false,
		IsWorn:      false,
		CanPickUp:   prototype.CanPickUp,
		Metadata:    make(map[string]string),
	}

	for k, v := range prototype.TraitMods {
		newItem.TraitMods[k] = v
	}

	for k, v := range prototype.Metadata {
		newItem.Metadata[k] = v
	}

	if newItem.Container {
		newItem.Contents = make([]*Item, 0, len(prototype.Contents))
		for _, contentItem := range prototype.Contents {
			newContentItem, err := s.CreateItemFromPrototype(contentItem.ID.String())
			if err != nil {
				log.Printf("Error creating content item from prototype %s: %v", contentItem.ID, err)
				continue
			}
			newItem.Contents = append(newItem.Contents, newContentItem)
		}
	}

	if err := s.Database.WriteItem(newItem); err != nil {
		return nil, fmt.Errorf("failed to write new item to database: %w", err)
	}

	log.Printf("Created new item %s (ID: %s) from prototype %s", newItem.Name, newItem.ID, prototypeID)
	return newItem, nil
}
