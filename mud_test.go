package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

// TestLoadConfigurationSuccess tests the successful loading of a configuration from a JSON file.
func TestLoadConfigurationSuccess(t *testing.T) {
	// Define the expected configuration based on the JSON structure.
	expected := Configuration{
		Port:           9050,
		UserPoolID:     "exampleId",
		ClientSecret:   "secret",
		UserPoolRegion: "us-east-1",
		ClientID:       "clientId",
		DataFile:       "data.json",
	}

	// Create a temporary JSON file with the configuration.
	tempFile, err := ioutil.TempFile("", "config-*.json")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up after the test.

	// Write the configuration to the temporary file.
	configContent := []byte(`{"Port":9050,"UserPoolId":"exampleId","UserPoolClientSecret":"secret","UserPoolRegion":"us-east-1","UserPoolClientId":"clientId","DataFile":"data.json"}`)
	if _, err := tempFile.Write(configContent); err != nil {
		t.Fatalf("Unable to write to temporary file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Unable to close temporary file: %v", err)
	}

	// Load the configuration from the file.
	config, err := loadConfiguration(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Compare the loaded configuration with the expected configuration.
	if !reflect.DeepEqual(config, expected) {
		t.Errorf("Expected configuration %+v, got %+v", expected, config)
	}
}

// TestLoadConfigurationFileNotFound tests the behavior when the configuration file does not exist.
func TestLoadConfigurationFileNotFound(t *testing.T) {
    _, err := loadConfiguration("nonExistentFile.json")
    if err == nil {
        t.Fatal("Expected an error for non-existent file, got nil")
    }
}

// TestLoadConfigurationInvalidJSON tests loading a configuration file with invalid JSON.
func TestLoadConfigurationInvalidJSON(t *testing.T) {
    tempFile, err := ioutil.TempFile("", "invalidConfig-*.json")
    if err != nil {
        t.Fatalf("Unable to create temporary file: %v", err)
    }
    defer os.Remove(tempFile.Name()) // Clean up after the test.

    invalidContent := []byte(`{Port:9050}`) // Incorrectly formatted JSON.
    if _, err := tempFile.Write(invalidContent); err != nil {
        t.Fatalf("Unable to write to temporary file: %v", err)
    }
    if err := tempFile.Close(); err != nil {
        t.Fatalf("Unable to close temporary file: %v", err)
    }

    _, err = loadConfiguration(tempFile.Name())
    if err == nil {
        t.Fatal("Expected an error for invalid JSON, got nil")
    }
}
