package main

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestServeThumbnailImageHandler(t *testing.T) {
	// Copy the test image to the uploads directory
	testImgSrc := "tests/images/test.jpg"
	testImgDst := "uploads/test.jpg"
	imgData, err := os.ReadFile(testImgSrc)
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}
	if err := os.WriteFile(testImgDst, imgData, 0644); err != nil {
		t.Fatalf("failed to copy test image: %v", err)
	}
	defer os.Remove(testImgDst)

	// Set up config for handler
	config.UploadPath = "uploads"

	// Create request and recorder
	req := httptest.NewRequest("GET", "/t/test.jpg", nil)
	rr := httptest.NewRecorder()

	serveThumbnailImageHandler(rr, req)

	resp := rr.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("expected Content-Type image/jpeg, got %s", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Errorf("response body is empty")
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to decode thumbnail image: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 568 || bounds.Dy() != 426 {
		t.Errorf("thumbnail image has dimensions %d,%d; want 568,426", bounds.Dx(), bounds.Dy())
	}
}

func TestShrinkImage(t *testing.T) {

	file, err := os.Open("tests/images/test.jpg")
	if err != nil {
		t.Fatalf("failed to open test image: %v", err)
	}
	defer file.Close()

	shrunk, format, err := shrinkImage(file, 4)
	if err != nil {
		t.Fatalf("shrinkImage failed: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("expected format 'jpeg', got '%s'", format)
	}

	bounds := shrunk.Bounds()
	if bounds.Dx() != 568 || bounds.Dy() != 426 {
		t.Errorf("shrunk image has dimensions %d,%d; want 568,426", bounds.Dx(), bounds.Dy())
	}

	// Make sure we've created a valid jpeg
	outFile, err := os.CreateTemp("", "shrunk_test_*.jpg")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(outFile.Name())
	defer outFile.Close()

	if err := jpeg.Encode(outFile, shrunk, &jpeg.Options{Quality: 85}); err != nil {
		t.Errorf("failed to encode shrunk image: %v", err)
	}
}
