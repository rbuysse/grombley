package main

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type MimeTypeHandler struct {
	mimeToExt map[string]string
	extToMime map[string]string
}

var supportedMimeTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/gif":  "gif",
}

func newMimeTypeHandler() *MimeTypeHandler {
	mimeToExt := supportedMimeTypes

	extToMime := make(map[string]string, len(mimeToExt))
	for mime, ext := range mimeToExt {
		extToMime["."+ext] = mime
		if mime == "image/jpeg" {
			extToMime[".jpeg"] = mime
		}
	}

	return &MimeTypeHandler{
		mimeToExt: mimeToExt,
		extToMime: extToMime,
	}
}

func (m *MimeTypeHandler) detectContentType(file io.Reader) (string, io.Reader, error) {
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", nil, err
	}

	contentType := http.DetectContentType(buffer[:n])
	combinedReader := io.MultiReader(bytes.NewReader(buffer[:n]), file)

	ext, ok := m.mimeToExt[contentType]
	if !ok {
		return "", nil, fmt.Errorf("unsupported type: %s", contentType)
	}

	return "." + ext, combinedReader, nil
}

func (m *MimeTypeHandler) getContentType(filename string) string {
	ext := filepath.Ext(filename)
	if mime, ok := m.extToMime[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func randfilename(length int, extension string) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomRunes := make([]rune, length)
	seed := rand.NewSource(time.Now().UnixNano())
	rand := rand.New(seed)
	for index := range randomRunes {
		randomRunes[index] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(randomRunes) + extension
}

func writeFileAndReturnURL(w http.ResponseWriter, r *http.Request, file io.Reader) error {
	// Buffer the entire file in memory so we can hash it and still use it
	fileData, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// Compute hash from buffered data
	hash, err := computeFileHash(bytes.NewReader(fileData))
	if err != nil {
		return err
	}
	value, exists := imageHashExists(hash)

	if exists {
		// Verify the file actually exists on disk
		filePath := filepath.Join(config.UploadPath, value)
		if _, err := os.Stat(filePath); err == nil {
			// File exists, return existing URL
			if config.Debug {
				fmt.Printf("Hash %s exists: %s\n", hash, value)
			}
			fileURL := constructFileURL(r, value)
			return respondWithFileURL(w, r, fileURL)
		} else {
			// File was deleted, remove stale hash and proceed with upload
			if config.Debug {
				fmt.Printf("Hash %s exists but file %s is missing, removing stale hash\n", hash, value)
			}
			removeHash(hash)
		}
	}

	if config.Debug {
		fmt.Printf("Hash %s does not exist\n", hash)
	}
	ext, fileReader, err := mimeTypeHandler.detectContentType(bytes.NewReader(fileData))
	if err != nil {
		http.Error(w, "Unsupported file type", http.StatusBadRequest)
		return err
	}

	genfilename := randfilename(6, ext)
	filepath := filepath.Join(config.UploadPath, genfilename)

	if err := processAndSaveImage(filepath, fileReader, ext); err != nil {
		http.Error(w, "Error processing file", http.StatusInternalServerError)
		return err
	}

	// Add hash to map (thread-safe)
	hashesMutex.Lock()
	hashes[hash] = genfilename
	hashesMutex.Unlock()

	fileURL := constructFileURL(r, genfilename)
	return respondWithFileURL(w, r, fileURL)
}

const uploadTempPrefix = ".upload-"

func createAndCopyFile(filename string, src io.Reader) error {
	// Publish only complete files so the startup scan cannot hash partial uploads.
	newFile, err := os.CreateTemp(filepath.Dir(filename), uploadTempPrefix)
	if err != nil {
		return fmt.Errorf("error creating the file: %w", err)
	}
	defer os.Remove(newFile.Name())
	defer newFile.Close()

	if _, err = io.Copy(newFile, src); err != nil {
		return fmt.Errorf("error copying file data: %w", err)
	}
	if err := newFile.Chmod(0644); err != nil {
		return fmt.Errorf("error setting file permissions: %w", err)
	}
	if err := newFile.Close(); err != nil {
		return fmt.Errorf("error closing the file: %w", err)
	}
	if err := os.Rename(newFile.Name(), filename); err != nil {
		return fmt.Errorf("error publishing the file: %w", err)
	}
	return nil
}

func processAndSaveImage(filepath string, src io.Reader, ext string) error {
	// For GIF files, just save as-is (animated GIFs shouldn't be re-encoded)
	if ext == ".gif" {
		return createAndCopyFile(filepath, src)
	}

	// Strip EXIF and metadata:
	data, err := stripExifButKeepOrientationFromReader(src)
	if err != nil {
		return fmt.Errorf("error processing image: %w", err)
	}

	// Write the processed image
	return createAndCopyFile(filepath, bytes.NewReader(data))
}

// validateImageName checks for path traversal, empty names, and allowed extensions
func validateImageName(imageName string, uploadPath string) error {
	if imageName == "" || imageName == "." || imageName == ".." {
		return fmt.Errorf("invalid file name")
	}

	if strings.HasPrefix(imageName, ".") {
		return fmt.Errorf("invalid file name")
	}

	imagePath := filepath.Join(uploadPath, imageName)
	absImagePath, err := filepath.Abs(imagePath)
	absUploadPath, err2 := filepath.Abs(uploadPath)
	if err != nil || err2 != nil || !strings.HasPrefix(absImagePath, absUploadPath) {
		return fmt.Errorf("invalid file path")
	}

	ext := strings.ToLower(filepath.Ext(imageName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		return nil
	default:
		return fmt.Errorf("unsupported file type")
	}
}
