package main

import (
	"log"

	bolt "github.com/etcd-io/bbolt"
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
