package main

import (
	"bytes"
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
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

type uploadTestReader func([]byte) (int, error)

func (read uploadTestReader) Read(p []byte) (int, error) {
	return read(p)
}

func TestBuildHashDictSkipsUnfinishedUpload(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "finished.gif")
	data := []byte("upload contents")
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	read := false
	go func() {
		done <- createAndCopyFile(filename, uploadTestReader(func(p []byte) (int, error) {
			if !read {
				read = true
				return copy(p, data), nil
			}
			close(started)
			<-release
			return 0, io.EOF
		}))
	}()
	select {
	case <-started:
	case err := <-done:
		t.Fatalf("upload ended before it could be scanned: %v", err)
	}
	defer func() {
		close(release)
		if err := <-done; err != nil {
			t.Errorf("upload failed: %v", err)
		}
	}()

	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		t.Errorf("unfinished upload is visible: %v", err)
	}
	built, err := buildHashDict(dir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(built) != 0 {
		t.Errorf("scan included unfinished upload: %v", built)
	}
}

func TestCreateAndCopyFilePublishesCompleteUpload(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "finished.gif")
	data := []byte("upload contents")
	if err := createAndCopyFile(filename, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("file contents = %q, want %q", got, data)
	}
	built, err := buildHashDict(dir)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := computeFileHash(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(built) != 1 || built[hash] != "finished.gif" {
		t.Errorf("scan did not find completed upload: %v", built)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("temporary file left after upload: %v", entries)
	}
}

func TestCreateAndCopyFileCleansUpFailedUpload(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "failed.gif")
	readErr := errors.New("upload interrupted")
	src := io.MultiReader(strings.NewReader("partial contents"), uploadTestReader(func([]byte) (int, error) {
		return 0, readErr
	}))
	if err := createAndCopyFile(filename, src); !errors.Is(err, readErr) {
		t.Fatalf("expected upload error, got %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("files left after failed upload: %v", entries)
	}
}
