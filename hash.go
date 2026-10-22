package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func buildHashDict(imageDir string) (map[string]string, error) {
	hashes := make(map[string]string)
	err := filepath.Walk(imageDir, func(path string, info os.FileInfo, err error) error {
		// A temporary upload may disappear when it is published during the walk.
		if strings.HasPrefix(filepath.Base(path), uploadTempPrefix) {
			return nil
		}
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

			hashes[hashString] = info.Name()
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

func imageHashExists(hash string) (string, bool) {
	hashesMutex.RLock()
	defer hashesMutex.RUnlock()
	if value, ok := hashes[hash]; ok {
		return value, true
	}
	return "", false
}

// loadHashes runs while the server is already handling requests, so it merges
// into the shared map under the mutex instead of replacing it. Entries added by
// uploads during the scan are kept.
func loadHashes(imageDir string) {
	built, err := buildHashDict(imageDir)
	if err != nil {
		fmt.Printf("Error building hash dict, only new uploads will be deduplicated: %v\n", err)
		return
	}

	hashesMutex.Lock()
	for hash, filename := range built {
		if _, ok := hashes[hash]; !ok {
			hashes[hash] = filename
		}
	}
	hashesMutex.Unlock()

	if config.Debug {
		for hash, filename := range built {
			fmt.Printf("MD5 Hash: %s, Filename: %s\n", hash, filename)
		}
	}
	fmt.Printf("Loaded %d image hashes\n", len(built))
}

// removeHash removes a hash entry from the map (thread-safe)
func removeHash(hash string) {
	hashesMutex.Lock()
	defer hashesMutex.Unlock()
	delete(hashes, hash)
}

// removeHashForFile removes hash entry for a given filename (thread-safe)
func removeHashForFile(filename string) {
	hashesMutex.Lock()
	defer hashesMutex.Unlock()
	for hash, name := range hashes {
		if name == filename {
			delete(hashes, hash)
			return
		}
	}
}
