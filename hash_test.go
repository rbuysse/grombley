package main

import (
	"fmt"
	"sync"
	"testing"
)

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
