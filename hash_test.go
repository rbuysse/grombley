package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBuildHashDictWithTimestamps verifies that buildHashDict:
// 1. Returns hash entries with filenames
// 2. Returns hash entries with modification timestamps
// 3. Timestamps are non-zero
func TestBuildHashDictWithTimestamps(t *testing.T) {
	// Create temp directory with test images
	tempDir := t.TempDir()

	// Copy test image
	testImage := "tests/images/test.jpg"
	data, err := os.ReadFile(testImage)
	if err != nil {
		t.Skip("Test image not found")
	}

	file1 := filepath.Join(tempDir, "image1.jpg")
	if err := os.WriteFile(file1, data, 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}

	// Wait a bit to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	file2 := filepath.Join(tempDir, "image2.jpg")
	if err := os.WriteFile(file2, data, 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	// Build hash dictionary
	hashDict, err := buildHashDict(tempDir)
	if err != nil {
		t.Fatalf("Failed to build hash dict: %v", err)
	}

	// Should have one entry (both files are identical)
	if len(hashDict) != 1 {
		t.Errorf("Expected 1 hash entry, got %d", len(hashDict))
	}

	// Get the hash entry
	var entry HashEntry
	for _, v := range hashDict {
		entry = v
		break
	}

	// Verify filename is set
	if entry.Filename == "" {
		t.Errorf("Filename is empty")
	}

	// Verify timestamp is set and non-zero
	if entry.ModTime.IsZero() {
		t.Errorf("ModTime is zero")
	}

	// Verify filename is one of our test files
	if entry.Filename != "image1.jpg" && entry.Filename != "image2.jpg" {
		t.Errorf("Unexpected filename: %s", entry.Filename)
	}

	t.Logf("Hash entry: filename=%s, modtime=%s", entry.Filename, entry.ModTime)
}

// TestImageHashExistsWithTimestamp verifies that imageHashExists:
// 1. Returns correct filename for existing hash
// 2. Returns correct timestamp for existing hash
// 3. Returns false for non-existing hash
func TestImageHashExistsWithTimestamp(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Read test image
	testImage := "tests/images/test.jpg"
	data, err := os.ReadFile(testImage)
	if err != nil {
		t.Skip("Test image not found")
	}

	// Write test file
	file1 := filepath.Join(tempDir, "test.jpg")
	if err := os.WriteFile(file1, data, 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Build hash dictionary
	hashDict, err := buildHashDict(tempDir)
	if err != nil {
		t.Fatalf("Failed to build hash dict: %v", err)
	}

	// Compute hash of the test image
	hash, err := computeFileHash(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	// Store the hash dict globally for imageHashExists to work
	oldHashes := hashes
	hashes = hashDict
	defer func() { hashes = oldHashes }()

	// Test existing hash
	entry, exists := imageHashExists(hash)
	if !exists {
		t.Errorf("Hash should exist")
	}

	if entry.Filename != "test.jpg" {
		t.Errorf("Expected filename 'test.jpg', got '%s'", entry.Filename)
	}

	if entry.ModTime.IsZero() {
		t.Errorf("ModTime should not be zero")
	}

	// Test non-existing hash
	_, exists = imageHashExists("nonexistent")
	if exists {
		t.Errorf("Hash should not exist")
	}

	t.Logf("Hash exists: filename=%s, modtime=%s", entry.Filename, entry.ModTime)
}

// TestHashEntryPreservesTimestamp verifies that when we add entries to the hash map,
// we preserve the file modification time
func TestHashEntryPreservesTimestamp(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Write test file with specific timestamp
	testFile := filepath.Join(tempDir, "test.jpg")
	testData := []byte("test image data")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Set specific modification time
	specificTime := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)
	if err := os.Chtimes(testFile, specificTime, specificTime); err != nil {
		t.Fatalf("Failed to set file times: %v", err)
	}

	// Build hash dictionary
	hashDict, err := buildHashDict(tempDir)
	if err != nil {
		t.Fatalf("Failed to build hash dict: %v", err)
	}

	// Get the entry
	var entry HashEntry
	for _, v := range hashDict {
		entry = v
		break
	}

	// Verify the timestamp matches what we set
	if !entry.ModTime.Equal(specificTime) {
		t.Errorf("Expected ModTime %s, got %s", specificTime, entry.ModTime)
	}
}
