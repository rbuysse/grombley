package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Port       string `toml:"port"`
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
	http.HandleFunc("/url", handleUrlUpload)
	http.HandleFunc("/foo", redirectHandler)
	http.HandleFunc("/fronc", froncHandler)
	http.HandleFunc("/", indexHandler)

	// Define the port you want the server to listen on
	port := config.Port // Change this to your desired port

	fmt.Printf("Server is running on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func froncHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "we fronc!\n")
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://piss.es/", http.StatusSeeOther)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templatesFolder, "templates/index.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl.Execute(w, nil)
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

	writeFileAndRedirect(w, r, file, handler.Filename)
}

func handleUrlUpload(w http.ResponseWriter, r *http.Request) {
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

	parts := strings.Split(parsedUrl.Path, "/")
	filename := parts[len(parts)-1]

	writeFileAndRedirect(w, r, resp.Body, filename)
}

func writeFileAndRedirect(w http.ResponseWriter, r *http.Request, file io.Reader, filename string) {
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

	http.Redirect(w, r, "/fronc", http.StatusSeeOther)
}

func randfilename(n int, f string) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	extension := strings.Split(f, ".")[1]
	return string(b) + "." + extension
}
