package main

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Serve thumbnail (1/4 size)
func serveThumbnailImageHandler(w http.ResponseWriter, r *http.Request) {
	imageName := filepath.Base(r.URL.Path)
	if err := validateImageName(imageName, config.UploadPath); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	imagePath := filepath.Join(config.UploadPath, imageName)
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		notfoundHandler(w)
		return
	}

	// Get the original orientation before shrinking
	orientation := getImageOrientation(imageData)

	// Shrink the image (this just decodes and shrinks - doesn't apply orientation)
	dst, format, err := shrinkImage(bytes.NewReader(imageData), 4)
	if err != nil {
		http.Error(w, "Failed to shrink image", http.StatusInternalServerError)
		return
	}

	// Encode the thumbnail
	var buf bytes.Buffer
	if format == "png" {
		err = png.Encode(&buf, dst)
	} else {
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85})
	}
	if err != nil {
		http.Error(w, "Failed to encode image", http.StatusInternalServerError)
		return
	}

	// Add the same orientation tag as the original so browsers display it correctly
	thumbnailData, _ := addOrientationTag(buf.Bytes(), orientation)

	if format == "png" {
		w.Header().Set("Content-Type", "image/png")
	} else {
		w.Header().Set("Content-Type", "image/jpeg")
	}
	w.Write(thumbnailData)
}

// shrinkImage reduces the size of an image by the given factor and returns the new image and format.
func shrinkImage(reader io.Reader, factor int) (image.Image, string, error) {
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, "", err
	}
	bounds := img.Bounds()
	newW := bounds.Dx() / factor
	newH := bounds.Dy() / factor
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := x * factor
			srcY := y * factor
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}
	return dst, format, nil
}
