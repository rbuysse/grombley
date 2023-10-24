package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
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

//go:embed templates
var templatesFolder embed.FS

func init() {
	_, err := toml.DecodeFile("config.toml", &config)
	if err != nil {
		log.Fatalf("Error in parsing config file: %v", err)
		os.Exit(1)
	}

	rand.Seed(time.Now().UnixNano())
}

func main() {
	// Create the upload directory if it doesn't exist
	if _, err := os.Stat(config.UploadPath); os.IsNotExist(err) {
		os.MkdirAll(config.UploadPath, os.ModePerm)
	}

	// Create a new HTTP router
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/url", urlUploadHandler)
	http.HandleFunc("/i/", serveImageHandler)
	http.HandleFunc("/", indexHandler)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Define the port you want the server to listen on
	port := config.Port // Change this to your desired port

	fmt.Printf("Server is running on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
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
		http.NotFound(w, r)
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
	file, handler, err := parseFormAndGetFile(r)
	if err != nil {
		handleError(w, "Error processing the file", err)
		return
	}

	err = checkFileType(file)
	if err != nil {
		handleError(w, "Invalid file type", err)
		return
	}

	writeFileAndRedirect(w, r, file, handler.Filename)
}

func urlUploadHandler(w http.ResponseWriter, r *http.Request) {
	urlString, err := parseRequestAndGetURL(r)
	if err != nil {
		handleError(w, "Error processing the URL", err)
		return
	}

	resp, err := http.Get(urlString)
	if err != nil {
		handleError(w, "Error retrieving file from URL", err)
		return
	}
	defer resp.Body.Close()

	filename := getFilenameFromURL(urlString)

	writeFileAndRedirect(w, r, resp.Body, filename)
}

func parseFormAndGetFile(r *http.Request) (multipart.File, *multipart.FileHeader, error) {
	r.ParseMultipartForm(100 << 20) // 100 MB max in-memory size
	return r.FormFile("file")
}

func parseRequestAndGetURL(r *http.Request) (string, error) {
	var requestBody map[string]string
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		return "", err
	}
	return requestBody["url"], nil
}

func getFilenameFromURL(urlString string) string {
	parsedUrl, _ := url.Parse(urlString)
	return path.Base(parsedUrl.Path)
}

func handleError(w http.ResponseWriter, message string, err error) {
	fmt.Println(message+":", err)
	http.Error(w, message, http.StatusBadRequest)
}

func checkFileType(file multipart.File) error {
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil {
		return err
	}
	filetype := http.DetectContentType(buffer)
	if !strings.HasPrefix(filetype, "image/") && !strings.HasPrefix(filetype, "video/") {
		return errors.New("Invalid file type: " + filetype)
	}
	file.Seek(0, io.SeekStart)
	return nil
}

func writeFileAndRedirect(w http.ResponseWriter, r *http.Request, file io.Reader, filename string) {
	filename = strings.ToLower(filename)
	genfilename := randfilename(6, filename)
	newFile, err := os.Create(config.UploadPath + genfilename)
	if err != nil {
		fmt.Println("Error creating the file:", err)
		http.Error(w, "Error creating the file", http.StatusInternalServerError)
		return
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, file)
	if err != nil {
		fmt.Println("Error copying file data:", err)
		http.Error(w, "Error copying file data", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/i/"+genfilename, http.StatusSeeOther)
}
