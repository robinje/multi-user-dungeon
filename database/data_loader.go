package main

import (
	"flag"
	"fmt"
	"log"
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

	// Initialize the rooms map
	rooms := make(map[int64]*Room)

	// Load data from BoltDB
	rooms, err := roomLoadBolt(rooms, *boltFilePath)
	if err != nil {
		fmt.Println("Room data load from BoltDB failed:", err)
	} else {
		fmt.Println("Room data loaded from BoltDB successfully")
	}

	// Load the JSON data
	rooms, err = roomLoadJSON(rooms, *jsonRoomFilePath)
	if err != nil {
		fmt.Println("Room data load failed:", err)
	} else {
		fmt.Println("Room data loaded successfully")
	}

	// Load JSON data from file
	archetypesData, err := archLoadJSON(*jsonArchFilePath)
	if err != nil {
		log.Fatalf("Failed to load Archetype JSON data: %v", err)
	}

	// Load JSON data from file
	prototypesData, err := protoLoadJSON(*jsonProtoFilePath)
	if err != nil {
		log.Fatalf("Failed to load Prototype JSON data: %v", err)
	}

	// Write data to BoltDB
	err = roomWriteBolt(rooms, *boltFilePath)
	if err != nil {
		fmt.Println("Room data write failed:", err)
		return // Ensure to exit if writing fails
	} else {
		fmt.Println("Room data written successfully")
	}

	// Store the data in BoltDB
	err = archWriteBolt(archetypesData, *boltFilePath)
	if err != nil {
		log.Fatalf("Failed to store Archetype data in BoltDB: %v", err)
	}

	fmt.Println("Archetype Data successfully stored in BoltDB.")

	// Store the data in BoltDB
	err = protoWriteBolt(prototypesData, *boltFilePath)
	if err != nil {
		log.Fatalf("Failed to store Prototype data in BoltDB: %v", err)
	}

	fmt.Println("Prototype Data successfully stored in BoltDB.")

	// Load data from BoltDB
	rooms, err = roomLoadBolt(rooms, *boltFilePath)
	if err != nil {
		fmt.Println("Room data load from BoltDB failed:", err)
	} else {
		fmt.Println("Room data loaded from BoltDB successfully")
	}

	// Load the data from BoltDB
	archetypesData, err = archLoadBolt(*boltFilePath)
	if err != nil {
		log.Fatalf("Failed to load Archetype data from BoltDB: %v", err)
	}

	// Load the data from BoltDB
	prototypesData, err = protoLoadBolt(*boltFilePath)
	if err != nil {
		log.Fatalf("Failed to load Prototype data from BoltDB: %v", err)
	}

	// Display the rooms
	roomDisplay(rooms)

	// Display the data
	archDisplay(archetypesData)

	// Display the data
	protoDisplay(prototypesData)
}
