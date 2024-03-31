package main

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temp file
	tempFile, err := os.CreateTemp("", "config")
	if err != nil {
		t.Fatalf("Error creating temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	configContent := `
port = ":666"
serve_path = "/p/"
upload_path = "./grapes/"
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Error writing to temporary file: %v", err)
	}

	config := loadConfig(tempFile.Name())

	// Check if the loaded config matches the expected config
	expectedPort := ":666"
	if config.Port != expectedPort {
		t.Errorf("Expected port to be %s, but got %s", expectedPort, config.Port)
	}
}
