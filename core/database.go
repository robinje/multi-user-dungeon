package core

import (
	"fmt"
	"log"
	"strings"

	bolt "go.etcd.io/bbolt"
)

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
	k.Mutex.Lock()
	defer k.Mutex.Unlock()

	err := k.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("create bucket %s: %w", bucketName, err)
		}
		if err := bucket.Put(key, value); err != nil {
			return fmt.Errorf("put key-value in bucket %s: %w", bucketName, err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("database update failed: %w", err)
	}

	return nil
}

func (k *KeyPair) Get(bucketName string, key []byte) ([]byte, error) {
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

func (k *KeyPair) ViewAllBuckets() error {
	return k.db.View(func(tx *bolt.Tx) error {
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
}

func (k *KeyPair) Shutdown() error {
	k.Mutex.Lock()
	defer k.Mutex.Unlock()

	if k.db != nil {
		if err := k.db.Sync(); err != nil {
			return fmt.Errorf("sync database: %w", err)
		}
		if err := k.db.Close(); err != nil {
			return fmt.Errorf("close database: %w", err)
		}
		k.db = nil
	}
	return nil
}
