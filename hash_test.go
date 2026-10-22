package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestLoadHashesKeepsEntriesAddedDuringScan(t *testing.T) {
	oldHashes := hashes
	defer func() { hashes = oldHashes }()

	imgData, err := os.ReadFile("tests/images/test.jpg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}
	pngData, err := os.ReadFile("tests/images/slimer.png")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "old.jpg"), imgData, 0644); err != nil {
		t.Fatalf("failed to write test image: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scanned.png"), pngData, 0644); err != nil {
		t.Fatalf("failed to write test image: %v", err)
	}
	jpgHash, _ := computeFileHash(bytes.NewReader(imgData))
	pngHash, _ := computeFileHash(bytes.NewReader(pngData))

	// Simulate uploads that landed while the scan was running
	hashes = map[string]string{jpgHash: "live.jpg", "abc123": "other.jpg"}

	loadHashes(dir)

	if got := hashes[jpgHash]; got != "live.jpg" {
		t.Errorf("expected entry added during scan to be kept, got %q", got)
	}
	if got := hashes["abc123"]; got != "other.jpg" {
		t.Errorf("expected unrelated entry to be kept, got %q", got)
	}
	if got := hashes[pngHash]; got != "scanned.png" {
		t.Errorf("expected scanned file to be added, got %q", got)
	}
}

func TestLoadHashesMissingDir(t *testing.T) {
	oldHashes := hashes
	hashes = map[string]string{"abc123": "foo.jpg"}
	defer func() { hashes = oldHashes }()

	// Should log and return rather than exit, so the server keeps serving
	loadHashes(filepath.Join(t.TempDir(), "nothere"))

	if got := hashes["abc123"]; got != "foo.jpg" {
		t.Errorf("expected existing hashes to be untouched, got %q", got)
	}
}

func TestRemoveHash(t *testing.T) {
	oldHashes := hashes
	hashes = map[string]string{"abc123": "foo.jpg", "def456": "bar.jpg"}
	defer func() { hashes = oldHashes }()

	removeHash("abc123")

	if _, ok := hashes["abc123"]; ok {
		t.Error("expected abc123 to be removed from hashes")
	}
	if _, ok := hashes["def456"]; !ok {
		t.Error("expected def456 to remain in hashes")
	}
}

func TestRemoveHashForFile(t *testing.T) {
	oldHashes := hashes
	hashes = map[string]string{"abc123": "foo.jpg", "def456": "bar.jpg"}
	defer func() { hashes = oldHashes }()

	removeHashForFile("foo.jpg")

	if _, ok := hashes["abc123"]; ok {
		t.Error("expected hash for foo.jpg (abc123) to be removed")
	}
	if _, ok := hashes["def456"]; !ok {
		t.Error("expected hash for bar.jpg (def456) to remain")
	}
}

func TestRemoveHashForFileNotFound(t *testing.T) {
	oldHashes := hashes
	hashes = map[string]string{"abc123": "foo.jpg"}
	defer func() { hashes = oldHashes }()

	// Should be a no-op if the filename isn't in the map
	removeHashForFile("nothere.jpg")

	if _, ok := hashes["abc123"]; !ok {
		t.Error("expected existing hash to be untouched after removing a non-existent filename")
	}
}

func TestConcurrentHashAccess(t *testing.T) {
	oldHashes := hashes
	hashes = make(map[string]string)
	defer func() { hashes = oldHashes }()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func(n int) {
			defer wg.Done()
			hashesMutex.Lock()
			hashes[fmt.Sprintf("hash%d", n)] = fmt.Sprintf("file%d.jpg", n)
			hashesMutex.Unlock()
		}(i)
		go func(n int) {
			defer wg.Done()
			imageHashExists(fmt.Sprintf("hash%d", n))
		}(i)
		go func(n int) {
			defer wg.Done()
			removeHashForFile(fmt.Sprintf("file%d.jpg", n))
		}(i)
	}
	wg.Wait()
}
