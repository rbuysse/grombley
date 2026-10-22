package main

import (
	"bytes"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func setupGalleryTest(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "grombley-gallery-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	oldConfig := config
	oldHashes := hashes
	config.UploadPath = dir
	config.ServePath = "/i/"
	hashes = make(map[string]string)
	mimeTypeHandler = *newMimeTypeHandler()
	return dir, func() {
		os.RemoveAll(dir)
		config = oldConfig
		hashes = oldHashes
	}
}

// TestWriteFileAndReturnURL_Dedup verifies that uploading the same image twice
// returns the same URL and only stores one file.
func TestWriteFileAndReturnURL_Dedup(t *testing.T) {
	_, cleanup := setupGalleryTest(t)
	defer cleanup()

	imgData, err := os.ReadFile("tests/images/test.jpg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	req1 := httptest.NewRequest("GET", "/", nil)
	rr1 := httptest.NewRecorder()
	if err := writeFileAndReturnURL(rr1, req1, bytes.NewReader(imgData)); err != nil {
		t.Fatalf("first upload failed: %v", err)
	}
	body1, _ := io.ReadAll(rr1.Result().Body)

	req2 := httptest.NewRequest("GET", "/", nil)
	rr2 := httptest.NewRecorder()
	if err := writeFileAndReturnURL(rr2, req2, bytes.NewReader(imgData)); err != nil {
		t.Fatalf("second upload of same image failed: %v", err)
	}
	body2, _ := io.ReadAll(rr2.Result().Body)

	if strings.TrimSpace(string(body1)) != strings.TrimSpace(string(body2)) {
		t.Errorf("expected dedup: same image should return same URL\n  first:  %s\n  second: %s", body1, body2)
	}

	entries, _ := os.ReadDir(config.UploadPath)
	if len(entries) != 1 {
		t.Errorf("expected 1 file after dedup, got %d", len(entries))
	}
}

// TestWriteFileAndReturnURL_StaleHash verifies that when a hash exists in memory
// but its file has been deleted from disk, the image is re-uploaded rather than
// returning a URL to the missing file.
func TestWriteFileAndReturnURL_StaleHash(t *testing.T) {
	dir, cleanup := setupGalleryTest(t)
	defer cleanup()

	imgData, err := os.ReadFile("tests/images/test.jpg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	// Compute the hash so we can seed a stale entry pointing to a non-existent file.
	hash, err := computeFileHash(bytes.NewReader(imgData))
	if err != nil {
		t.Fatalf("failed to compute hash: %v", err)
	}
	hashes[hash] = "deleted.jpg"

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	if err := writeFileAndReturnURL(rr, req, bytes.NewReader(imgData)); err != nil {
		t.Fatalf("upload with stale hash failed: %v", err)
	}

	// A new file should have been created on disk.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 new file in upload dir, got %d", len(entries))
	}
	newFilename := entries[0].Name()

	// The hash map should now point to the new file, not the deleted one.
	if got := hashes[hash]; got != newFilename {
		t.Errorf("expected hashes[%s] = %q, got %q", hash, newFilename, got)
	}

	// The response URL should reference the new file, not the deleted one.
	body, _ := io.ReadAll(rr.Result().Body)
	if strings.Contains(string(body), "deleted.jpg") {
		t.Errorf("response URL references the deleted file: %s", body)
	}
	if !strings.Contains(string(body), newFilename) {
		t.Errorf("response URL should reference the new file %q: %s", newFilename, body)
	}
}
