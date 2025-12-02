package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// HashEntry stores metadata about a hashed file
type HashEntry struct {
	Filename string
	ModTime  time.Time
}

func buildHashDict(imageDir string) (map[string]HashEntry, error) {
	hashes := make(map[string]HashEntry)
	err := filepath.Walk(imageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			hash := md5.New()
			if _, err := io.Copy(hash, file); err != nil {
				return err
			}
			hashInBytes := hash.Sum(nil)[:16]
			hashString := fmt.Sprintf("%x", hashInBytes)

			hashes[hashString] = HashEntry{
				Filename: info.Name(),
				ModTime:  info.ModTime(),
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking the path %q: %v", imageDir, err)
	}
	return hashes, nil
}

func computeFileHash(fileReader io.Reader) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, fileReader); err != nil {
		return "", err
	}
	hashInBytes := hash.Sum(nil)[:16]
	hashString := fmt.Sprintf("%x", hashInBytes)

	if seeker, ok := fileReader.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			return "", err
		}
	}
	return hashString, nil
}

func imageHashExists(hash string) (HashEntry, bool) {
	if entry, ok := hashes[hash]; ok {
		return entry, true
	}
	return HashEntry{}, false
}
