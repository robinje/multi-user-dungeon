package core

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
)

// DisplayPrototypes logs the details of each prototype for debugging purposes.
func DisplayPrototypes(prototypes *PrototypesData) {
	for _, prototype := range prototypes.ItemPrototypes {
		Logger.Debug("Prototype", "name", prototype.Name, "description", prototype.Description)
	}
}

// LoadPrototypesFromJSON loads item prototypes from a JSON file.
func LoadPrototypesFromJSON(fileName string) (*PrototypesData, error) {
	file, err := os.ReadFile(fileName)
	if err != nil {
		Logger.Error("Error reading prototypes JSON file", "fileName", fileName, "error", err)
		return nil, fmt.Errorf("error reading prototypes JSON file: %w", err)
	}

	var data PrototypesData
	err = json.Unmarshal(file, &data)
	if err != nil {
		Logger.Error("Error unmarshalling prototypes JSON data", "fileName", fileName, "error", err)
		return nil, fmt.Errorf("error unmarshalling prototypes JSON data: %w", err)
	}

	for _, prototype := range data.ItemPrototypes {
		Logger.Debug("Loaded prototype from JSON", "name", prototype.Name, "description", prototype.Description)
	}

	return &data, nil
}

// StorePrototypes stores item prototypes into the DynamoDB table.
func (kp *KeyPair) StorePrototypes(prototypes *PrototypesData) error {
	for _, prototype := range prototypes.ItemPrototypes {
		// Ensure the prototype has an ID
		if prototype.ID == "" {
			prototype.ID = uuid.New().String()
		}

		// Use the updated Put method which includes the key within the item.
		err := kp.Put("prototypes", prototype)
		if err != nil {
			Logger.Error("Error storing prototype", "name", prototype.Name, "error", err)
			return fmt.Errorf("error storing prototype %s: %w", prototype.Name, err)
		}

		Logger.Info("Stored prototype", "name", prototype.Name, "prototypeID", prototype.ID)
	}

	return nil
}

// LoadPrototypes retrieves all item prototypes from the DynamoDB table.
func (kp *KeyPair) LoadPrototypes() (*PrototypesData, error) {
	prototypesData := &PrototypesData{}

	var itemPrototypes []ItemData
	err := kp.Scan("prototypes", &itemPrototypes)
	if err != nil {
		Logger.Error("Error scanning prototypes table", "error", err)
		return nil, fmt.Errorf("error scanning prototypes: %w", err)
	}

	prototypesData.ItemPrototypes = itemPrototypes

	for _, prototype := range prototypesData.ItemPrototypes {
		Logger.Debug("Loaded prototype from database", "name", prototype.Name, "description", prototype.Description)
	}

	return prototypesData, nil
}

// LoadItem retrieves an item or prototype from the DynamoDB table.
func (k *KeyPair) LoadItem(id string, isPrototype bool) (*Item, error) {
	if id == "" {
		return nil, fmt.Errorf("empty item ID provided")
	}

	tableName := "items"
	if isPrototype {
		tableName = "prototypes"
	}

	key := map[string]*dynamodb.AttributeValue{
		"ItemID": {
			S: aws.String(id),
		},
	}

	var itemData ItemData
	err := k.Get(tableName, key, &itemData)
	if err != nil {
		Logger.Error("Error loading item data", "itemID", id, "tableName", tableName, "error", err)
		return nil, fmt.Errorf("error loading item data: %w", err)
	}

	itemID, err := uuid.Parse(itemData.ID)
	if err != nil {
		Logger.Error("Error parsing item UUID", "itemID", itemData.ID, "error", err)
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
		Mutex:       sync.Mutex{},
	}

	// Load contents if the item is a container
	if item.Container {
		item.Contents = make([]*Item, 0, len(itemData.Contents))
		for _, contentID := range itemData.Contents {
			contentItem, err := k.LoadItem(contentID, false)
			if err != nil {
				Logger.Error("Error loading content item", "contentID", contentID, "parentItemID", id, "error", err)
				continue // Skip this content item but continue loading others
			}
			item.Contents = append(item.Contents, contentItem)
		}
	}

	Logger.Info("Successfully loaded item", "itemName", item.Name, "itemID", item.ID, "tableName", tableName)
	return item, nil
}

// WriteItem stores an item into the DynamoDB table, handling nested contents if it's a container.
func (k *KeyPair) WriteItem(obj *Item) error {
	// Recursively write contained items if the item is a container
	if obj.Container {
		for _, contentItem := range obj.Contents {
			if err := k.WriteItem(contentItem); err != nil {
				Logger.Error("Error writing content item", "contentItemID", contentItem.ID, "parentItemID", obj.ID, "error", err)
				return fmt.Errorf("error writing content item %s: %w", contentItem.ID, err)
			}
		}
	}

	// Prepare the list of content IDs
	contentIDs := make([]string, 0, len(obj.Contents))
	for _, contentItem := range obj.Contents {
		contentIDs = append(contentIDs, contentItem.ID.String())
	}

	// Create the ItemData struct to store in DynamoDB
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

	// Determine the table name based on whether the item is a prototype
	tableName := "items"
	if obj.IsPrototype {
		tableName = "prototypes"
	}

	// Write the item data to the DynamoDB table
	err := k.Put(tableName, itemData)
	if err != nil {
		Logger.Error("Error writing item data", "itemName", obj.Name, "itemID", obj.ID, "tableName", tableName, "error", err)
		return fmt.Errorf("error writing item data: %w", err)
	}

	Logger.Info("Successfully wrote item", "itemName", obj.Name, "itemID", obj.ID, "tableName", tableName)
	return nil
}

// SaveActiveItems saves all active items from rooms and characters to the database.
func (s *Server) SaveActiveItems() error {
	if s == nil {
		return fmt.Errorf("server is nil")
	}

	Logger.Info("Starting to save active items...")

	// Collect all items from rooms and characters
	itemsToSave := make(map[uuid.UUID]*Item)

	// Items in rooms
	if s.Rooms != nil {
		for roomID, room := range s.Rooms {
			if room == nil {
				Logger.Warn("Nil room found", "roomID", roomID)
				continue
			}
			room.Mutex.Lock()
			for _, item := range room.Items {
				if item == nil {
					Logger.Warn("Nil item found in room", "roomID", roomID)
					continue
				}
				itemsToSave[item.ID] = item
			}
			room.Mutex.Unlock()
		}
	} else {
		Logger.Warn("Server Rooms map is nil")
	}

	// Items in character inventories
	if s.Characters != nil {
		for charID, character := range s.Characters {
			if character == nil {
				Logger.Warn("Nil character found", "characterID", charID)
				continue
			}
			character.Mutex.Lock()
			for _, item := range character.Inventory {
				if item == nil {
					Logger.Warn("Nil item found in inventory", "characterID", charID)
					continue
				}
				itemsToSave[item.ID] = item
			}
			character.Mutex.Unlock()
		}
	} else {
		Logger.Warn("Server Characters map is nil")
	}

	// Save all collected items
	if s.Database == nil {
		return fmt.Errorf("server database is nil")
	}

	for _, item := range itemsToSave {
		if item == nil {
			Logger.Warn("Attempting to save a nil item, skipping")
			continue
		}
		if err := s.Database.WriteItem(item); err != nil {
			Logger.Error("Error saving item", "itemName", item.Name, "itemID", item.ID, "error", err)
			// Continue saving other items even if one fails
		} else {
			Logger.Info("Successfully saved item", "itemName", item.Name, "itemID", item.ID)
		}
	}

	Logger.Info("Finished saving active items")
	return nil
}

// CreateItemFromPrototype creates a new item instance from a prototype.
func (s *Server) CreateItemFromPrototype(prototypeID string) (*Item, error) {
	prototype, err := s.Database.LoadItem(prototypeID, true)
	if err != nil {
		Logger.Error("Failed to load item prototype", "prototypeID", prototypeID, "error", err)
		return nil, fmt.Errorf("failed to load item prototype: %w", err)
	}

	if !prototype.IsPrototype {
		Logger.Error("Item is not a prototype", "itemID", prototypeID)
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
		Mutex:       sync.Mutex{},
	}

	// Copy trait modifications and metadata
	for k, v := range prototype.TraitMods {
		newItem.TraitMods[k] = v
	}

	for k, v := range prototype.Metadata {
		newItem.Metadata[k] = v
	}

	// Recursively create contents if the item is a container
	if newItem.Container {
		newItem.Contents = make([]*Item, 0, len(prototype.Contents))
		for _, contentItem := range prototype.Contents {
			newContentItem, err := s.CreateItemFromPrototype(contentItem.ID.String())
			if err != nil {
				Logger.Error("Error creating content item from prototype", "prototypeID", contentItem.ID.String(), "error", err)
				continue // Skip this content item but continue with others
			}
			newItem.Contents = append(newItem.Contents, newContentItem)
		}
	}

	// Save the new item to the database
	if err := s.Database.WriteItem(newItem); err != nil {
		Logger.Error("Failed to write new item to database", "itemName", newItem.Name, "itemID", newItem.ID, "error", err)
		return nil, fmt.Errorf("failed to write new item to database: %w", err)
	}

	Logger.Info("Created new item from prototype", "itemName", newItem.Name, "itemID", newItem.ID, "prototypeID", prototypeID)
	return newItem, nil
}

// itemFromData creates an Item from ItemData
func (kp *KeyPair) itemFromData(itemData *ItemData) (*Item, error) {
	if itemData == nil {
		return nil, fmt.Errorf("itemData is nil")
	}

	itemID, err := uuid.Parse(itemData.ID)
	if err != nil {
		return nil, fmt.Errorf("error parsing item UUID: %w", err)
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
		Mutex:       sync.Mutex{},
	}

	// Handle Contents if the item is a container
	if item.Container {
		item.Contents = make([]*Item, 0, len(itemData.Contents))
		for _, contentID := range itemData.Contents {
			contentItem, err := kp.LoadItem(contentID, false) // Assuming LoadItem is accessible here
			if err != nil {
				Logger.Error("Error loading content item", "contentID", contentID, "parentItemID", item.ID, "error", err)
				continue // Skip this content item but continue with others
			}
			item.Contents = append(item.Contents, contentItem)
		}
	}

	return item, nil
}
