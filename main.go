package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed templates
var templatesFolder embed.FS

const (
	uploadPath    = "./uploads/"    // Change this to your desired upload folder path
	maxUploadSize = 5 * 1024 * 1024 // 5 MB, adjust as needed
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	// Create the upload directory if it doesn't exist
	if _, err := os.Stat(uploadPath); os.IsNotExist(err) {
		os.MkdirAll(uploadPath, os.ModePerm)
	}

	// Create a new HTTP router
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/foo", redirectHandler)
	http.HandleFunc("/fronc", froncHandler)
	http.HandleFunc("/", indexHandler)

	// Define the port you want the server to listen on
	port := ":8080" // Change this to your desired port

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

	// Create or open a new file in the desired directory
	// Replace "/path/to/your/directory" with your actual directory path
	genfilename := randfilename(6, handler.Filename)
	newFile, err := os.Create("./uploads/" + genfilename)
	if err != nil {
		fmt.Println("Error creating the file:", err)
		http.Error(w, "Error creating the file", http.StatusInternalServerError)
		return
	}
	defer newFile.Close()

	// Copy the uploaded file data to the new file
	_, err = io.Copy(newFile, file)
	if err != nil {
		fmt.Println("Error copying file data:", err)
		http.Error(w, "Error copying file data", http.StatusInternalServerError)
		return
	}

	// fmt.Fprintln(w, "File uploaded successfully:", handler.Filename)
	// http.Redirect(w, r, "http://localhost:8080/" + genfilename, http.StatusSeeOther)
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
