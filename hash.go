package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func buildHashDict(imageDir string) (map[string]string, error) {
	hashes := make(map[string]string)
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
	if value, ok := hashes[hash]; ok {
		return value, true
	}
	return "", false
}
