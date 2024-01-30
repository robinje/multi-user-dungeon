package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"sync"

	bolt "go.etcd.io/bbolt"
)

type DataBase struct {
	File string
	db   *bolt.DB
}

func (b *DataBase) Open() error {
	var err error
	b.db, err = bolt.Open(b.File, 0600, nil)
	if err != nil {
		return err
	}
	return nil
}

// Put saves a key-value pair to the database.
func (b *DataBase) Put(bucketName string, key, value []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		return bucket.Put(key, value)
	})
}

// Get retrieves a value for a given key from the database.
func (b *DataBase) Get(bucketName string, key []byte) ([]byte, error) {
	var value []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return bolt.ErrBucketNotFound
		}
		value = bucket.Get(key)
		return nil
	})
	return value, err
}

// Delete removes a key-value pair from the database.
func (b *DataBase) Delete(bucketName string, key []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return bolt.ErrBucketNotFound
		}
		return bucket.Delete(key)
	})
}

// CloseDB closes the open database.
func (b *DataBase) Close() {
	if b.db != nil {
		err := b.db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
}

type Index struct {
	IndexID uint64
	mu      sync.Mutex
}

func (i *Index) GetID() uint64 {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.IndexID++
	return i.IndexID
}

func (i *Index) SetID(id uint64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if id > i.IndexID {
		i.IndexID = id
	}
}

// Initialize updates the IndexID to the highest index in the specified bucket.
func (i *Index) Initialize(db *bolt.DB, bucketName string) error {
	highestIndex, err := FindHighestIndex(db, bucketName)
	if err != nil {
		return err
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	i.IndexID = highestIndex + 1
	return nil
}

// FindHighestIndex finds the highest index in a specified bucket of a BoltDB database.
func FindHighestIndex(db *bolt.DB, bucketName string) (uint64, error) {
	var highestIndex uint64 = 0

	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}

		cursor := bucket.Cursor()
		for k, _ := cursor.Last(); k != nil; k, _ = cursor.Prev() {
			// Assuming the keys are stored as big-endian encoded integers
			keyIndex := uint64(binary.BigEndian.Uint64(k))
			if keyIndex > highestIndex {
				highestIndex = keyIndex
			}
			break // In this case, we break after finding the first (highest) index
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return highestIndex, nil
}
