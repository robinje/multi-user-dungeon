package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/robinje/multi-user-dungeon/core"
)

func main() {
	jsonRoomFilePath := flag.String("r", "test_rooms.json", "Path to the Rooms JSON file.")
	jsonArchFilePath := flag.String("a", "test_archetypes.json", "Path to the Archetypes JSON file.")
	jsonProtoFilePath := flag.String("p", "test_prototypes.json", "Path to the Prototypes JSON file.")
	boltFilePath := flag.String("b", "test_data.bolt", "Path to the Bolt DB file.")
	help := flag.Bool("h", false, "Display help.")

	flag.Parse()

	if *help {
		fmt.Println("Usage of program:")
		fmt.Println("  -r string")
		fmt.Println("        Path to the Rooms JSON file. (default \"test_rooms.json\")")
		fmt.Println("  -a string")
		fmt.Println("        Path to the Archetypes JSON file. (default \"test_archetypes.json\")")
		fmt.Println("  -p string")
		fmt.Println("        Path to the Prototypes JSON file. (default \"test_prototypes.json\")")
		fmt.Println("  -b string")
		fmt.Println("        Path to the Bolt DB file. (default \"test_data.bolt\")")
		fmt.Println("  -h")
		fmt.Println("        Display help.")
		return
	}

	// Initialize the database connection
	kp, err := core.NewKeyPair(*boltFilePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer kp.Close()

	// Load rooms from JSON
	rooms, err := core.LoadRoomsFromJSON(*jsonRoomFilePath)
	if err != nil {
		log.Printf("Room data load from JSON failed: %v", err)
	} else {
		fmt.Println("Room data loaded from JSON successfully")

		// Store rooms in the database
		err = kp.StoreRooms(rooms)
		if err != nil {
			log.Printf("Failed to store rooms in BoltDB: %v", err)
		} else {
			fmt.Println("Room data stored in BoltDB successfully")
		}
	}

	// Load archetypes from JSON and store in BoltDB
	archetypes, err := core.LoadArchetypesFromJSON(*jsonArchFilePath)
	if err != nil {
		log.Printf("Failed to load Archetype JSON data: %v", err)
	} else {
		fmt.Println("Archetypes loaded from JSON successfully")
		err = kp.StoreArchetypes(archetypes)
		if err != nil {
			log.Printf("Failed to store Archetypes in BoltDB: %v", err)
		} else {
			fmt.Println("Archetype data stored in BoltDB successfully")
		}
	}

	// Load prototypes from JSON and store in BoltDB
	prototypes, err := core.LoadPrototypesFromJSON(*jsonProtoFilePath)
	if err != nil {
		log.Printf("Failed to load Prototype JSON data: %v", err)
	} else {
		fmt.Println("Prototypes loaded from JSON successfully")
		err = kp.StorePrototypes(prototypes)
		if err != nil {
			log.Printf("Failed to store Prototypes in BoltDB: %v", err)
		} else {
			fmt.Println("Prototype data stored in BoltDB successfully")
		}
	}

	// Load data from BoltDB
	loadedRooms, err := kp.LoadRooms()
	if err != nil {
		log.Printf("Room data load from BoltDB failed: %v", err)
	} else {
		fmt.Println("Room data loaded from BoltDB successfully")
		core.DisplayRooms(loadedRooms)
	}

	loadedArchetypes, err := kp.LoadArchetypes()
	if err != nil {
		log.Printf("Failed to load Archetype data from BoltDB: %v", err)
	} else {
		fmt.Println("Archetype data loaded from BoltDB successfully")
		core.DisplayArchetypes(loadedArchetypes)
	}

	loadedPrototypes, err := kp.LoadPrototypes()
	if err != nil {
		log.Printf("Failed to load Prototype data from BoltDB: %v", err)
	} else {
		fmt.Println("Prototype data loaded from BoltDB successfully")
		core.DisplayPrototypes(loadedPrototypes)
	}
}
