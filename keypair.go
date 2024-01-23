package main

import (
	"log"

	"github.com/etcd-io/bbolt"
)

var db *bbolt.DB

func OpenDB(path string) error {
	var err error
	db, err = bbolt.Open(path, 0600, nil)
	if err != nil {
		return err
	}
	return nil
}

// Put saves a key-value pair to the database.
func Put(bucketName string, key, value []byte) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		return b.Put(key, value)
	})
}

// Get retrieves a value for a given key from the database.
func Get(bucketName string, key []byte) ([]byte, error) {
	var value []byte
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return bbolt.ErrBucketNotFound
		}
		value = b.Get(key)
		return nil
	})
	return value, err
}

// Delete removes a key-value pair from the database.
func Delete(bucketName string, key []byte) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return bbolt.ErrBucketNotFound
		}
		return b.Delete(key)
	})
}

// CloseDB closes the open database.
func CloseDB() {
	if db != nil {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
}
