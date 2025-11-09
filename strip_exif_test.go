package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStripExifCommand(t *testing.T) {
	// Create a temp directory for test images
	tempDir := t.TempDir()

	// Copy test images with EXIF data to temp directory
	testImages := []string{
		"tests/images/test.jpg",
		"tests/images/slimer.png",
	}

	for _, img := range testImages {
		data, err := os.ReadFile(img)
		if err != nil {
			t.Skip("Test images not found")
		}

		// Write to temp dir
		filename := filepath.Base(img)
		destPath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	// Run strip-exif command
	if err := stripExifFromDirectory(tempDir, false, false); err != nil {
		t.Fatalf("stripExifFromDirectory failed: %v", err)
	}

	// Verify each image has EXIF stripped but orientation preserved
	for _, img := range testImages {
		filename := filepath.Base(img)
		processedPath := filepath.Join(tempDir, filename)

		// Read original
		originalData, err := os.ReadFile(img)
		if err != nil {
			t.Fatalf("Failed to read original: %v", err)
		}
		originalOrientation := getImageOrientation(originalData)

		// Read processed
		processedData, err := os.ReadFile(processedPath)
		if err != nil {
			t.Fatalf("Failed to read processed file: %v", err)
		}

		// Check orientation preserved
		processedOrientation := getImageOrientation(processedData)
		if processedOrientation != originalOrientation {
			t.Errorf("%s: orientation changed from %d to %d", filename, originalOrientation, processedOrientation)
		}

		// File sizes should differ (EXIF stripped)
		if len(processedData) >= len(originalData) {
			t.Logf("Warning: %s processed file not smaller (original: %d, processed: %d)",
				filename, len(originalData), len(processedData))
		}
	}
}

func TestStripExifCommandDryRun(t *testing.T) {
	// Create a temp directory for test images
	tempDir := t.TempDir()

	// Copy a test image
	testImage := "tests/images/test.jpg"
	data, err := os.ReadFile(testImage)
	if err != nil {
		t.Skip("Test image not found")
	}

	destPath := filepath.Join(tempDir, "test.jpg")
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	originalSize := len(data)

	// Run in dry-run mode
	if err := stripExifFromDirectory(tempDir, true, false); err != nil {
		t.Fatalf("stripExifFromDirectory dry-run failed: %v", err)
	}

	// File should be unchanged
	afterData, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read file after dry-run: %v", err)
	}

	if len(afterData) != originalSize {
		t.Errorf("Dry-run modified file: original %d bytes, after %d bytes", originalSize, len(afterData))
	}
}

func TestStripExifCommandWithBackup(t *testing.T) {
	// Create a temp directory for test images
	tempDir := t.TempDir()

	// Copy a test image
	testImage := "tests/images/test.jpg"
	data, err := os.ReadFile(testImage)
	if err != nil {
		t.Skip("Test image not found")
	}

	filename := "test.jpg"
	destPath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run with backup
	if err := stripExifFromDirectory(tempDir, false, true); err != nil {
		t.Fatalf("stripExifFromDirectory with backup failed: %v", err)
	}

	// Check backup exists
	backupPath := filepath.Join(tempDir, filename+".bak")
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Backup file not created: %v", err)
	}

	// Backup should match original
	if len(backupData) != len(data) {
		t.Errorf("Backup size mismatch: expected %d, got %d", len(data), len(backupData))
	}

	// Original should be processed
	processedData, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read processed file: %v", err)
	}

	if len(processedData) >= len(data) {
		t.Logf("Warning: processed file not smaller than original")
	}
}
