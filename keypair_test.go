package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func setupTestDB(t *testing.T) (*KeyPair, func()) {
	t.Helper()

	// Create a temporary file for the database
	tmpfile, err := ioutil.TempFile("", "testdb_*.db")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	dbPath := tmpfile.Name()

	// Ensure the file is closed and removed after the test
	cleanup := func() {
		tmpfile.Close()
		os.Remove(dbPath)
	}

	// Initialize KeyPair with the temporary database file
	kp, err := NewKeyPair(dbPath)
	if err != nil {
		t.Fatalf("Failed to create KeyPair: %v", err)
	}

	return kp, cleanup
}

func TestNewKeyPair(t *testing.T) {
	kp, cleanup := setupTestDB(t)
	defer cleanup()

	if kp == nil {
		t.Errorf("Expected KeyPair instance, got nil")
	}
}

func TestPutAndGet(t *testing.T) {
	kp, cleanup := setupTestDB(t)
	defer cleanup()

	bucketName := "testBucket"
	key := []byte("testKey")
	value := []byte("testValue")

	if err := kp.Put(bucketName, key, value); err != nil {
		t.Fatalf("Failed to put value: %v", err)
	}

	retrievedValue, err := kp.Get(bucketName, key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if !reflect.DeepEqual(retrievedValue, value) {
		t.Errorf("Expected value %s, got %s", value, retrievedValue)
	}
}

func TestDelete(t *testing.T) {
	kp, cleanup := setupTestDB(t)
	defer cleanup()

	bucketName := "testBucket"
	key := []byte("testKey")
	value := []byte("testValue")

	// Put a value to then delete
	if err := kp.Put(bucketName, key, value); err != nil {
		t.Fatalf("Failed to put value: %v", err)
	}

	// Delete the value
	if err := kp.Delete(bucketName, key); err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	// Attempt to retrieve the deleted value
	retrievedValue, err := kp.Get(bucketName, key)
	if err != nil {
		t.Fatalf("Unexpected error retrieving key after delete: %v", err)
	}

	if retrievedValue != nil {
		t.Errorf("Expected nil value for deleted key, got %v", retrievedValue)
	}
}

func TestNextIndex(t *testing.T) {
	kp, cleanup := setupTestDB(t)
	defer cleanup()

	bucketName := "testBucket"

	index, err := kp.NextIndex(bucketName)
	if err != nil {
		t.Fatalf("Failed to get next index: %v", err)
	}

	if index == 0 {
		t.Errorf("Expected index to be greater than 0, got %d", index)
	}
}
