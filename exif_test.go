package main

import (
	"bytes"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

func TestImageOrientationPreserved(t *testing.T) {
	testCases := []struct {
		name string
		file string
	}{
		{"PNG", "tests/images/slimer.png"},
		{"JPEG", "tests/images/test.jpg"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Read test image
			data, err := os.ReadFile(tc.file)
			if err != nil {
				t.Skip("Test image not found")
			}

			// Get original orientation
			originalOrientation := getImageOrientation(data)
			t.Logf("Original orientation: %d", originalOrientation)

			// Process the image
			processed, err := stripExifButKeepOrientation(data)
			if err != nil {
				t.Fatalf("Error processing image: %v", err)
			}

			// Get processed orientation - should always match original
			processedOrientation := getImageOrientation(processed)
			if processedOrientation != originalOrientation {
				t.Errorf("Orientation mismatch: expected %d, got %d", originalOrientation, processedOrientation)
			} else {
				t.Logf("Orientation preserved: %d", processedOrientation)
			}
		})
	}
}

func TestThumbnailOrientationPreserved(t *testing.T) {
	// Read test image with orientation
	data, err := os.ReadFile("tests/images/slimer.png")
	if err != nil {
		t.Skip("Test image not found")
	}

	// Get original orientation (could be 1 or any other value)
	originalOrientation := getImageOrientation(data)
	t.Logf("Original orientation: %d", originalOrientation)

	// Simulate thumbnail generation
	dst, format, err := shrinkImage(bytes.NewReader(data), 4)
	if err != nil {
		t.Fatalf("Failed to shrink image: %v", err)
	}

	// Encode thumbnail
	var buf bytes.Buffer
	if format == "png" {
		err = png.Encode(&buf, dst)
	} else {
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85})
	}
	if err != nil {
		t.Fatalf("Failed to encode thumbnail: %v", err)
	}

	// Add orientation tag
	thumbnailData, err := addOrientationTag(buf.Bytes(), originalOrientation)
	if err != nil {
		t.Fatalf("Failed to add orientation tag: %v", err)
	}

	// Check orientation in thumbnail
	thumbnailOrientation := getImageOrientation(thumbnailData)
	if thumbnailOrientation != originalOrientation {
		t.Errorf("Thumbnail orientation mismatch: expected %d, got %d", originalOrientation, thumbnailOrientation)
	} else {
		t.Logf("Thumbnail orientation preserved: %d", thumbnailOrientation)
	}
}
