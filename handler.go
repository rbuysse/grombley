package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
)

func notfoundHandler(w http.ResponseWriter) {
	tmpl, err := template.ParseFS(templatesFolder, "templates/404.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl.Execute(w, nil)
}

func livezHandler(w http.ResponseWriter, req *http.Request) {
	_, verbose := req.URL.Query()["verbose"]
	if !verbose {
		fmt.Fprintf(w, "200")
		return
	}
	// Print extra info if verbose is present http://foo.bar:3000/livez?verbose
	fmt.Fprintf(w, "Server is running on http://%s\n", config.Bind)
	fmt.Fprintf(w, "Serving images at %s\n", config.ServePath)
	fmt.Fprintf(w, "Upload path is %s\n", config.UploadPath)
	fmt.Fprintf(w, "%d image hashes in memory\n", len(hashes))
}

func readyzHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "200")
}

// Serve original image
func serveImageHandler(w http.ResponseWriter, r *http.Request) {
	imageName := filepath.Base(r.URL.Path)

	if err := validateImageName(imageName, config.UploadPath); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Construct the full path to the image file.
	imagePath := filepath.Join(config.UploadPath, imageName)

	// Open the image file.
	imageFile, err := os.Open(imagePath)
	if err != nil {
		notfoundHandler(w)
		return
	}
	defer imageFile.Close()

	// Set the Content-Type header based on the file extension.
	contentType := mimeTypeHandler.getContentType(imageName)
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
	file, _, err := r.FormFile("file") // "file" should match the name attribute in your HTML form
	if err != nil {
		fmt.Println("Error retrieving the file:", err)
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	writeFileAndReturnURL(w, r, file)
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

	writeFileAndReturnURL(w, r, resp.Body)
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
