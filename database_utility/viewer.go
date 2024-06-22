package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	bolt "go.etcd.io/bbolt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run script.go <path_to_database>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	fmt.Printf("Contents of Bbolt database: %s\n", dbPath)
	fmt.Println(strings.Repeat("=", 50))

	err = db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			fmt.Printf("\nBucket: %s\n", name)
			fmt.Println(strings.Repeat("-", 30))

			count := 0
			err := b.ForEach(func(k, v []byte) error {
				fmt.Printf("  Key:   %s\n", k)
				fmt.Printf("  Value: %s\n", v)
				fmt.Println()
				count++
				return nil
			})

			fmt.Printf("Total entries in bucket: %d\n", count)
			return err
		})
	})

	if err != nil {
		log.Fatalf("Error reading database: %v", err)
	}
}
