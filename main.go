package main

import (
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
)

type Config struct {
	Bind       string `toml:"bind"`
	Debug      bool   `toml:"debug"`
	ServePath  string `toml:"serve_path"`
	UploadPath string `toml:"upload_path"`
}

var config Config
var hashes map[string]string
var mimeTypeHandler MimeTypeHandler

//go:embed templates
var templatesFolder embed.FS

func main() {

	config = GenerateConfig()
	mimeTypeHandler = *newMimeTypeHandler()

	// Create the upload directory if it doesn't exist
	if _, err := os.Stat(config.UploadPath); os.IsNotExist(err) {
		fmt.Printf("Creating upload directory at %s\n", config.UploadPath)
		os.MkdirAll(config.UploadPath, os.ModePerm)
	}
	var err error

	hashesChan := make(chan map[string]string)
	errChan := make(chan error)

	hashes, err = buildHashDict(config.UploadPath)
	go func() {
		hashes, err := buildHashDict(config.UploadPath)
		if err != nil {
			errChan <- err
			return
		}
		hashesChan <- hashes
	}()

	// Create a new HTTP router
	http.HandleFunc("/livez", livezHandler)
	http.HandleFunc("/readyz", readyzHandler)
	http.HandleFunc("/t/", serveThumbnailImageHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/url", urlUploadHandler)
	http.HandleFunc(config.ServePath, serveImageHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		filePath := path.Join("templates", r.URL.Path)
		if r.URL.Path == "/" {
			filePath = "templates/index.html"
		}
		file, err := templatesFolder.Open(filePath)
		if err != nil {
			notfoundHandler(w)
			return
		}
		defer file.Close()

		io.Copy(w, file)
	})

	if config.Debug {
		fmt.Println("Debug mode is enabled")
	}

	fmt.Printf("Server is running on http://%s\n"+
		"Serving images at %s\n"+
		"Upload path is %s\n",

		config.Bind, config.ServePath, config.UploadPath)

	select {
	case hashes = <-hashesChan:
	case err = <-errChan:
		fmt.Printf("Error: %v\n", err)
		return
	}

	if config.Debug {
		for hash, filename := range hashes {
			fmt.Printf("MD5 Hash: %s, Filename: %s\n", hash, filename)
		}
	}

	log.Fatal(http.ListenAndServe(config.Bind, nil))
}
