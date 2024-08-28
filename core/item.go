package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
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
	for _, prototype := range prototypes.ItemPrototypes {
		av, err := dynamodbattribute.MarshalMap(prototype)
		if err != nil {
			return fmt.Errorf("error marshalling prototype %s: %w", prototype.Name, err)
		}

		key := map[string]*dynamodb.AttributeValue{
			"ID": {S: aws.String(prototype.ID)},
		}

		err = kp.Put("prototypes", key, av)
		if err != nil {
			return fmt.Errorf("error storing prototype %s: %w", prototype.Name, err)
		}

		fmt.Printf("Stored prototype: %s\n", prototype.Name)
	}

	return nil
}

func (kp *KeyPair) LoadPrototypes() (*PrototypesData, error) {
	prototypesData := &PrototypesData{}

	var itemPrototypes []ItemData
	err := kp.Scan("prototypes", &itemPrototypes)
	if err != nil {
		return nil, fmt.Errorf("error scanning prototypes: %w", err)
	}

	prototypesData.ItemPrototypes = itemPrototypes

	for _, prototype := range prototypesData.ItemPrototypes {
		fmt.Printf("Loaded prototype: %s - %s\n", prototype.Name, prototype.Description)
	}

	return prototypesData, nil
}

func (k *KeyPair) LoadItem(id string, isPrototype bool) (*Item, error) {
	tableName := "items"
	if isPrototype {
		tableName = "prototypes"
	}

	key := map[string]*dynamodb.AttributeValue{
		"ID": {S: aws.String(id)},
	}

	var itemData ItemData
	err := k.Get(tableName, key, &itemData)
	if err != nil {
		return nil, fmt.Errorf("error loading item data: %w", err)
	}

	itemID, err := uuid.Parse(itemData.ID)
	if err != nil {
		return nil, fmt.Errorf("error parsing item ID: %w", err)
	}

	item := &Item{
		ID:          itemID,
		Name:        itemData.Name,
		Description: itemData.Description,
		Mass:        itemData.Mass,
		Value:       itemData.Value,
		Stackable:   itemData.Stackable,
		MaxStack:    itemData.MaxStack,
		Quantity:    itemData.Quantity,
		Wearable:    itemData.Wearable,
		WornOn:      itemData.WornOn,
		Verbs:       itemData.Verbs,
		Overrides:   itemData.Overrides,
		TraitMods:   itemData.TraitMods,
		Container:   itemData.Container,
		IsPrototype: itemData.IsPrototype,
		IsWorn:      itemData.IsWorn,
		CanPickUp:   itemData.CanPickUp,
		Metadata:    itemData.Metadata,
	}

	if item.Container {
		item.Contents = make([]*Item, 0, len(itemData.Contents))
		for _, contentID := range itemData.Contents {
			contentItem, err := k.LoadItem(contentID, false)
			if err != nil {
				return nil, fmt.Errorf("error loading content item %s: %w", contentID, err)
			}
			item.Contents = append(item.Contents, contentItem)
		}
	}

	return item, nil
}

func (k *KeyPair) WriteItem(obj *Item) error {
	contentIDs := make([]string, 0, len(obj.Contents))
	for _, contentItem := range obj.Contents {
		contentIDs = append(contentIDs, contentItem.ID.String())
		// Recursively write contained items
		if err := k.WriteItem(contentItem); err != nil {
			return fmt.Errorf("error writing content item %s: %w", contentItem.ID, err)
		}
	}

	itemData := ItemData{
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

	av, err := dynamodbattribute.MarshalMap(itemData)
	if err != nil {
		return fmt.Errorf("error marshalling item data: %w", err)
	}

	tableName := "items"
	if obj.IsPrototype {
		tableName = "prototypes"
	}

	key := map[string]*dynamodb.AttributeValue{
		"ID": {S: aws.String(obj.ID.String())},
	}

	err = k.Put(tableName, key, av)
	if err != nil {
		return fmt.Errorf("error writing item data: %w", err)
	}

	return nil
}

func (s *Server) SaveActiveItems() error {
	if s == nil {
		return fmt.Errorf("server is nil")
	}

	log.Println("Starting to save active items...")

	// Collect all items from rooms and characters
	itemsToSave := make(map[uuid.UUID]*Item)

	// Items in rooms
	if s.Rooms != nil {
		for roomID, room := range s.Rooms {
			if room == nil {
				log.Printf("Warning: Nil room found with ID %d", roomID)
				continue
			}
			room.Mutex.Lock()
			for itemID, item := range room.Items {
				if item == nil {
					log.Printf("Warning: Nil item found in room %d with ID %s", roomID, itemID)
					continue
				}
				itemsToSave[item.ID] = item
			}
			room.Mutex.Unlock()
		}
	} else {
		log.Println("Warning: Server Rooms map is nil")
	}

	// Items in character inventories
	if s.Characters != nil {
		for charName, character := range s.Characters {
			if character == nil {
				log.Printf("Warning: Nil character found with name %s", charName)
				continue
			}
			character.Mutex.Lock()
			for itemName, item := range character.Inventory {
				if item == nil {
					log.Printf("Warning: Nil item found in inventory of character %s with name %s", charName, itemName)
					continue
				}
				itemsToSave[item.ID] = item
			}
			character.Mutex.Unlock()
		}
	} else {
		log.Println("Warning: Server Characters map is nil")
	}

	// Save all collected items
	if s.Database == nil {
		return fmt.Errorf("server database is nil")
	}

	for _, item := range itemsToSave {
		if item == nil {
			log.Println("Warning: Attempting to save a nil item, skipping")
			continue
		}
		if err := s.Database.WriteItem(item); err != nil {
			log.Printf("Error saving item %s (ID: %s): %v", item.Name, item.ID, err)
			// Continue saving other items even if one fails
		} else {
			log.Printf("Successfully saved item %s (ID: %s)", item.Name, item.ID)
		}
	}

	log.Println("Finished saving active items")
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

func (r *Room) AddItem(item *Item) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if item == nil {
		log.Printf("Warning: Attempted to add nil item to room %d", r.RoomID)
		return
	}

	if r.Items == nil {
		r.Items = make(map[string]*Item)
	}

	r.Items[item.ID.String()] = item
	log.Printf("Added item %s (ID: %s) to room %d", item.Name, item.ID, r.RoomID)
}

func (r *Room) RemoveItem(item *Item) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if item == nil {
		log.Printf("Warning: Attempted to remove nil item from room %d", r.RoomID)
		return
	}

	delete(r.Items, item.ID.String())
	log.Printf("Removed item %s (ID: %s) from room %d", item.Name, item.ID, r.RoomID)
}

// Add a new method to clean up nil items
func (r *Room) CleanupNilItems() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	for id, item := range r.Items {
		if item == nil {
			delete(r.Items, id)
			log.Printf("Removed nil item with ID %s from room %d", id, r.RoomID)
		}
	}
}
