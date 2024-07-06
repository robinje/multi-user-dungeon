package main

import (
	"fmt"
	"log"
	"sync"

	bolt "go.etcd.io/bbolt"
)

type KeyPair struct {
	db    *bolt.DB
	file  string
	Mutex sync.Mutex // Mutex to synchronize write access
}

func NewKeyPair(file string) (*KeyPair, error) {

	log.Printf("Opening database %s", file)

	db, err := bolt.Open(file, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	return &KeyPair{db: db, file: file}, nil
}

func (k *KeyPair) Close() {

	log.Printf("Closing database %s", k.file)

	k.Mutex.Lock() // Ensure we synchronize close operations
	defer k.Mutex.Unlock()

	if k.db != nil {
		err := k.db.Close()
		if err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
}

func (k *KeyPair) Put(bucketName string, key, value []byte) error {

	log.Printf("Putting key %s in bucket %s", key, bucketName)

	k.Mutex.Lock() // Lock for write operation
	defer k.Mutex.Unlock()

	return k.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		return bucket.Put(key, value)
	})
}

func (k *KeyPair) Get(bucketName string, key []byte) ([]byte, error) {

	log.Printf("Getting key %s from bucket %s", key, bucketName)

	// No need to lock for read operations; BoltDB supports concurrent reads.
	var value []byte
	err := k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return bolt.ErrBucketNotFound
		}
		value = bucket.Get(key)
		return nil
	})
	return value, err
}

func (k *KeyPair) Delete(bucketName string, key []byte) error {

	log.Printf("Deleting key %s from bucket %s", key, bucketName)

	k.Mutex.Lock() // Lock for write operation
	defer k.Mutex.Unlock()

	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return bolt.ErrBucketNotFound
		}
		return bucket.Delete(key)
	})
}

func (k *KeyPair) NextIndex(bucketName string) (uint64, error) {

	log.Printf("Getting next index for bucket %s", bucketName)

	k.Mutex.Lock() // Lock for write operation
	defer k.Mutex.Unlock()

	var nextIndex uint64
	err := k.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("error creating or retrieving bucket: %w", err)
		}

		nextIndex, err = bucket.NextSequence()
		if err != nil {
			return fmt.Errorf("error getting next sequence number: %w", err)
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return nextIndex, nil
}
