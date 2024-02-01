package main

import (
	"log"
	"sync"

	bolt "go.etcd.io/bbolt"
)

type KeyPair struct {
	db   *bolt.DB
	file string
}

func NewKeyPair(file string) (*KeyPair, error) {
	db, err := bolt.Open(file, 0600, nil)
	if err != nil {
		log.Fatal("Error opening database: ", err)
		return nil, err
	}

	return &KeyPair{db: db, file: file}, nil
}

func (k *KeyPair) Close() {
	if k.db != nil {
		err := k.db.Close()
		if err != nil {
			log.Fatal("Error closing database: ", err)
		}
	}
}

func (k *KeyPair) Put(bucketName string, key, value []byte) error {
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		return bucket.Put(key, value)
	})
}

func (k *KeyPair) Get(bucketName string, key []byte) ([]byte, error) {
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
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return bolt.ErrBucketNotFound
		}
		return bucket.Delete(key)
	})
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
