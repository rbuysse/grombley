package main

import (
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestMakeThumb(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "thumbtest")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	config.UploadPath = tempDir

	// Create thumbs subdir
	err = os.Mkdir(filepath.Join(tempDir, "thumbs"), 0755)
	if err != nil {
		t.Fatalf("failed to create thumbs dir: %v", err)
	}

	// Copy test image into tempDir
	srcImgPath := "tests/images/test.jpg"
	dstImgPath := filepath.Join(tempDir, "test.jpg")
	srcImg, err := os.Open(srcImgPath)
	if err != nil {
		t.Fatalf("failed to open source image: %v", err)
	}
	defer srcImg.Close()
	dstImg, err := os.Create(dstImgPath)
	if err != nil {
		t.Fatalf("failed to create dest image: %v", err)
	}
	if _, err := io.Copy(dstImg, srcImg); err != nil {
		t.Fatalf("failed to copy image: %v", err)
	}
	dstImg.Close()

	thumbPath := filepath.Join(tempDir, "thumbs", "test.jpg")

	err = makeThumb(dstImgPath)
	if err != nil {
		t.Fatalf("makeThumb failed: %v", err)
	}

	f, err := os.Open(thumbPath)
	if err != nil {
		t.Fatalf("thumbnail not created: %v", err)
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		t.Fatalf("failed to decode thumbnail: %v", err)
	}

	if img.Bounds().Dx() <= 0 || img.Bounds().Dy() <= 0 {
		t.Errorf("thumbnail has invalid dimensions: %v", img.Bounds())
	}

	origFile, err := os.Open(dstImgPath)
	if err != nil {
		t.Fatalf("failed to open original: %v", err)
	}
	defer origFile.Close()
	origImg, _, err := image.Decode(origFile)
	if err != nil {
		t.Fatalf("failed to decode original: %v", err)
	}
	if img.Bounds().Dx() != origImg.Bounds().Dx()/4 || img.Bounds().Dy() != origImg.Bounds().Dy()/4 {
		t.Errorf("thumbnail size incorrect: got %v, want %v", img.Bounds(), image.Rect(0, 0, origImg.Bounds().Dx()/4, origImg.Bounds().Dy()/4))
	}
}
