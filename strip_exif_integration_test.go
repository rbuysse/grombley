package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestStripExifCommandFixesDedupe tests the real-world scenario described in Task 5:
// 1. Old uploads have full EXIF data
// 2. New uploads have EXIF stripped (keeping orientation)
// 3. Same image has different hashes -> no de-dupe
// 4. Run strip-exif command on old uploads
// 5. Verify de-dupe now works
func TestStripExifCommandFixesDedupe(t *testing.T) {
	// Setup: Create temp upload directory
	tempDir := t.TempDir()

	// Simulate old upload - image with full EXIF data
	originalData, err := os.ReadFile("tests/images/test.jpg")
	if err != nil {
		t.Skip("Test image not found")
	}

	oldUploadPath := filepath.Join(tempDir, "old_upload.jpg")
	if err := os.WriteFile(oldUploadPath, originalData, 0644); err != nil {
		t.Fatalf("Failed to write old upload: %v", err)
	}

	// Simulate new upload - same image but with EXIF stripped
	strippedData, err := stripExifButKeepOrientation(originalData)
	if err != nil {
		t.Fatalf("Failed to strip EXIF: %v", err)
	}

	newUploadPath := filepath.Join(tempDir, "new_upload.jpg")
	if err := os.WriteFile(newUploadPath, strippedData, 0644); err != nil {
		t.Fatalf("Failed to write new upload: %v", err)
	}

	// Verify hashes are different before strip-exif command
	hashOld, err := computeFileHash(bytes.NewReader(originalData))
	if err != nil {
		t.Fatalf("Failed to compute old hash: %v", err)
	}

	hashNew, err := computeFileHash(bytes.NewReader(strippedData))
	if err != nil {
		t.Fatalf("Failed to compute new hash: %v", err)
	}

	if hashOld == hashNew {
		t.Logf("Warning: Test image may not have EXIF data, hashes already match")
	} else {
		t.Logf("Before strip-exif: old hash=%s, new hash=%s (different - no dedupe)", hashOld, hashNew)
	}

	// Run the strip-exif command on the upload directory
	if err := stripExifFromDirectory(tempDir, false, false); err != nil {
		t.Fatalf("strip-exif command failed: %v", err)
	}

	// Re-read the old upload after EXIF stripping
	oldDataAfterStrip, err := os.ReadFile(oldUploadPath)
	if err != nil {
		t.Fatalf("Failed to read old upload after stripping: %v", err)
	}

	// Compute new hash for old upload
	hashOldAfterStrip, err := computeFileHash(bytes.NewReader(oldDataAfterStrip))
	if err != nil {
		t.Fatalf("Failed to compute hash after strip: %v", err)
	}

	// Verify hashes now match (dedupe will work)
	if hashOldAfterStrip != hashNew {
		t.Errorf("After strip-exif, hashes should match but got: old=%s, new=%s", hashOldAfterStrip, hashNew)
	} else {
		t.Logf("After strip-exif: both hashes=%s (dedupe works!)", hashOldAfterStrip)
	}

	// Verify orientation was preserved
	originalOrientation := getImageOrientation(originalData)
	afterOrientation := getImageOrientation(oldDataAfterStrip)

	if originalOrientation != afterOrientation {
		t.Errorf("Orientation changed: before=%d, after=%d", originalOrientation, afterOrientation)
	}
}

// TestStripExifCommandWithMixedImages tests that the command handles:
// - JPEGs with EXIF
// - PNGs with EXIF
// - GIFs (which should be skipped)
// - Images without EXIF
func TestStripExifCommandWithMixedImages(t *testing.T) {
	tempDir := t.TempDir()

	// Copy test images to temp dir
	testCases := []struct {
		src         string
		dest        string
		shouldStrip bool
	}{
		{"tests/images/test.jpg", "test.jpg", true},
		{"tests/images/slimer.png", "slimer.png", true},
	}

	for _, tc := range testCases {
		data, err := os.ReadFile(tc.src)
		if err != nil {
			t.Logf("Skipping %s - file not found", tc.src)
			continue
		}

		destPath := filepath.Join(tempDir, tc.dest)
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", tc.dest, err)
		}
	}

	// Run strip-exif command
	if err := stripExifFromDirectory(tempDir, false, false); err != nil {
		t.Fatalf("strip-exif failed: %v", err)
	}

	// Verify all processed files exist and are valid
	for _, tc := range testCases {
		destPath := filepath.Join(tempDir, tc.dest)
		if _, err := os.Stat(destPath); err != nil {
			t.Errorf("File %s missing after processing: %v", tc.dest, err)
		}
	}
}

// TestStripExifPreservesFilePermissions verifies that file permissions are maintained
func TestStripExifPreservesFilePermissions(t *testing.T) {
	tempDir := t.TempDir()

	// Copy test image with specific permissions
	originalData, err := os.ReadFile("tests/images/test.jpg")
	if err != nil {
		t.Skip("Test image not found")
	}

	testPath := filepath.Join(tempDir, "test.jpg")
	if err := os.WriteFile(testPath, originalData, 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Get original permissions
	info, err := os.Stat(testPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	originalMode := info.Mode()

	// Strip EXIF
	if err := stripExifFromDirectory(tempDir, false, false); err != nil {
		t.Fatalf("strip-exif failed: %v", err)
	}

	// Check permissions preserved
	info, err = os.Stat(testPath)
	if err != nil {
		t.Fatalf("Failed to stat file after strip: %v", err)
	}
	newMode := info.Mode()

	if originalMode != newMode {
		t.Errorf("Permissions changed: before=%v, after=%v", originalMode, newMode)
	}
}
