package main

import (
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
)

type Config struct {
	Bind       string `toml:"bind"`
	Debug      bool   `toml:"debug"`
	ServePath  string `toml:"serve_path"`
	UploadPath string `toml:"upload_path"`
}

var config Config
var hashes = make(map[string]string)
var hashesMutex sync.RWMutex
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
	// Hash existing uploads in the background so the server starts listening
	// right away instead of after a full scan of the upload directory. This is
	// safe because every read and write of hashes goes through hashesMutex.
	// The only cost: an upload matching a file the scan hasn't reached yet is
	// saved as a duplicate instead of being deduplicated.
	go loadHashes(config.UploadPath)

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

	log.Fatal(http.ListenAndServe(config.Bind, nil))
}
