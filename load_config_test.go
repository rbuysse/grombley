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
bind = "localhost:666"
serve_path = "/p/"
upload_path = "./grapes/"
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Error writing to temporary file: %v", err)
	}

	config := loadConfig(tempFile.Name())

	// Check if the loaded config matches the expected config
	expectedBind := "localhost:666"
	if config.Bind != expectedBind {
		t.Errorf("Expected bind to be %s, but got %s", expectedBind, config.Bind)
	}

	expectedServePath := "/p/"
	if config.ServePath != expectedServePath {
		t.Errorf("Expected serve_path to be %s, but got %s", expectedServePath, config.ServePath)
	}

	expectedUploadPath := "./grapes/"
	if config.UploadPath != expectedUploadPath {
		t.Errorf("Expected upload_path to be %s, but got %s", expectedUploadPath, config.UploadPath)
	}

	// Test merging logic with some values missing
	tempFile2, err := os.CreateTemp("", "config-partial")
	if err != nil {
		t.Fatalf("Error creating temporary file: %v", err)
	}

	defer os.Remove(tempFile2.Name())
	partialConfigContent := `
	bind = "localhost:777"
	`
	if _, err := tempFile2.Write([]byte(partialConfigContent)); err != nil {
		t.Fatalf("Error writing to temporary file: %v", err)
	}

	config = loadConfig(tempFile2.Name())

	// Check if the loaded config merges correctly
	expectedBind = "localhost:777"
	if config.Bind != expectedBind {
		t.Errorf("Expected bind to be %s, but got %s", expectedBind, config.Bind)
	}

	expectedServePath = "/i/"
	if config.ServePath != expectedServePath {
		t.Errorf("Expected serve_path to be %s, but got %s", expectedServePath, config.ServePath)
	}

	expectedUploadPath = "./uploads/"
	if config.UploadPath != expectedUploadPath {
		t.Errorf("Expected upload_path to be %s, but got %s", expectedUploadPath, config.UploadPath)
	}
}
