package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Port       string `toml:"port"`
	ServePath  string `toml:"serve_path"`
	UploadPath string `toml:"upload_path"`
}

var config Config

const usage = `Usage:
  -c, --config         Path to a configuration file (default: config.toml)
`

//go:embed templates
var templatesFolder embed.FS

func init() {

	rand.Seed(time.Now().UnixNano())
}

func main() {

	// Parse the command-line flags and load the config
	var configFile string

	flag.StringVar(&configFile, "c", "config.toml", "Path to the configuration file")
	flag.StringVar(&configFile, "config", "config.toml", "Path to the configuration file")

	if !flag.Parsed() {
		flag.Parse()
	}

	// Load the config file if it exists otherwise use default values
	if _, err := os.Stat(configFile); err != nil {
		config.Port = "3000"
		config.ServePath = "/i/"
		config.UploadPath = "./uploads/"
	} else {
		config = loadConfig(configFile)
	}

	// Create the upload directory if it doesn't exist
	if _, err := os.Stat(config.UploadPath); os.IsNotExist(err) {
		os.MkdirAll(config.UploadPath, os.ModePerm)
	}

	// Create a new HTTP router
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/url", urlUploadHandler)
	http.HandleFunc(config.ServePath, serveImageHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			indexHandler(w, r)
		} else {
			notfoundHandler(w, r)
		}
	})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	config.Port = strings.TrimPrefix(config.Port, ":")

	fmt.Printf("Server is running on port %s\n", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

func loadConfig(configFile string) Config {

	var config Config

	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Fatalf("Error in parsing config file: %v", err)
		os.Exit(1)
	}

	return config
}

func getContentType(filename string) string {
	switch filepath.Ext(filename) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

func notfoundHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templatesFolder, "templates/404.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl.Execute(w, nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templatesFolder, "templates/index.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl.Execute(w, nil)
}

func randfilename(length int, filename string) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomRunes := make([]rune, length)
	for index := range randomRunes {
		randomRunes[index] = letterRunes[rand.Intn(len(letterRunes))]
	}
	extension := path.Ext(filename)
	return string(randomRunes) + extension
}

func serveImageHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the requested image filename from the URL.
	imageName := filepath.Base(r.URL.Path)

	// Construct the full path to the image file.
	imagePath := filepath.Join(config.UploadPath, imageName)

	// Open the image file.
	imageFile, err := os.Open(imagePath)
	if err != nil {
		notfoundHandler(w, r)
		return
	}
	defer imageFile.Close()

	// Set the Content-Type header based on the file extension.
	contentType := getContentType(imageName)
	w.Header().Set("Content-Type", contentType)

	// Copy the file data to the response writer.
	_, err = io.Copy(w, imageFile)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart form data with a specified max memory limit (in bytes)
	r.ParseMultipartForm(10 << 20) // 10 MB max in-memory size

	// Get the uploaded file
	file, handler, err := r.FormFile("file") // "file" should match the name attribute in your HTML form
	if err != nil {
		fmt.Println("Error retrieving the file:", err)
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	writeFileAndReturnURL(w, r, file, handler.Filename)
}

func urlUploadHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody map[string]string
	json.NewDecoder(r.Body).Decode(&requestBody)
	urlString := requestBody["url"]

	resp, err := http.Get(urlString)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	parsedUrl, err := url.Parse(urlString)
	if err != nil {
		return
	}

	filename := path.Base(parsedUrl.Path)

	writeFileAndReturnURL(w, r, resp.Body, filename)
}

func writeFileAndReturnURL(w http.ResponseWriter, r *http.Request, file io.Reader, filename string) error {
	filename = strings.ToLower(filename)
	genfilename := randfilename(6, filename)
	filepath := filepath.Join(config.UploadPath, genfilename)

	if err := createAndCopyFile(filepath, file); err != nil {
		fmt.Printf("%s: %v\n", "Error processing file", err)
		http.Error(w, "Error processing file", http.StatusInternalServerError)
		return err
	}

	fileURL := constructFileURL(r, genfilename)
	return respondWithFileURL(w, r, fileURL)
}

func createAndCopyFile(filepath string, src io.Reader) error {
	newFile, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("error creating the file: %w", err)
	}
	defer newFile.Close()

	if _, err = io.Copy(newFile, src); err != nil {
		return fmt.Errorf("error copying file data: %w", err)
	}

	return nil
}

func constructFileURL(r *http.Request, filename string) string {
	scheme := "http://"
	if r.TLS != nil {
		scheme = "https://"
	}
	return fmt.Sprintf("%s%s%s%s", scheme, r.Host, config.ServePath, filename)
}

func respondWithFileURL(w http.ResponseWriter, r *http.Request, url string) error {
	acceptHeader := r.Header.Get("Accept")
	switch acceptHeader {
	case "application/json":
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]string{"url": url})
		if err != nil {
			http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
			return err
		}
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, err := w.Write([]byte(url + "\n"))
		if err != nil {
			http.Error(w, "Failed to write plain text response", http.StatusInternalServerError)
			return err
		}
	}
	return nil
}
