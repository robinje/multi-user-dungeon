package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/robinje/multi-user-dungeon/core"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run viewer.go <path_to_database>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	kp, err := core.NewKeyPair(dbPath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer kp.Close()

	fmt.Printf("Contents of Bbolt database: %s\n", dbPath)
	fmt.Println(strings.Repeat("=", 50))

	err = kp.ViewAllBuckets()
	if err != nil {
		log.Fatalf("Error reading database: %v", err)
	}
}
