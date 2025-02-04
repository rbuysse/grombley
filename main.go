package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"text/template"
	"time"
)

type Config struct {
	Bind       string `toml:"bind"`
	Debug      bool   `toml:"debug"`
	Thumbs     bool   `toml:"thumbs"`
	ServePath  string `toml:"serve_path"`
	UploadPath string `toml:"upload_path"`
}

type MimeTypeHandler struct {
	mimeToExt map[string]string
	extToMime map[string]string
}

var config Config
var hashes map[string]string
var mimeTypeHandler MimeTypeHandler

//go:embed templates
var templatesFolder embed.FS

var supportedMimeTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/gif":  "gif",
}

func main() {

	config = GenerateConfig()
	mimeTypeHandler = *newMimeTypeHandler()

	// Create the upload directory if it doesn't exist
	if _, err := os.Stat(config.UploadPath); os.IsNotExist(err) {
		fmt.Printf("Creating upload directory at %s\n", config.UploadPath)
		os.MkdirAll(config.UploadPath, os.ModePerm)
	}
	if config.Thumbs {
		if _, err := os.Stat(config.UploadPath + "/thumbs"); os.IsNotExist(err) {
			fmt.Printf("Creating thumbnail directory at %s\n", config.UploadPath+"/thumbs")
			os.MkdirAll(config.UploadPath+"/thumbs", os.ModePerm)
		}
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

		if config.Thumbs {
			for _, filename := range hashes {
				thumbPath := filepath.Join(config.UploadPath, "thumbs", filepath.Base(filename))
				if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
					if err := makeThumb(filepath.Join(config.UploadPath, filename)); err != nil {
						fmt.Printf("Error creating thumbnail for %s: %v\n", filename, err)
					}
				}
			}
		}
	}()

	// Create a new HTTP router
	http.HandleFunc("/livez", livezHandler)
	http.HandleFunc("/readyz", readyzHandler)
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

	fmt.Printf("Server is running on http://%s\n"+
		"Serving images at %s\n"+
		"Upload path is %s\n"+
		"Thumbs is %t\n",
		config.Bind, config.ServePath, config.UploadPath, config.Thumbs)

	select {
	case hashes = <-hashesChan:
	case err = <-errChan:
		fmt.Printf("Error: %v\n", err)
		return
	}

	if config.Debug {
		fmt.Printf("config: %+v\n", config)
		for hash, filename := range hashes {
			fmt.Printf("MD5 Hash: %s, Filename: %s\n", hash, filename)
		}
	}

	log.Fatal(http.ListenAndServe(config.Bind, nil))
}

func newMimeTypeHandler() *MimeTypeHandler {
	mimeToExt := supportedMimeTypes

	extToMime := make(map[string]string, len(mimeToExt))
	for mime, ext := range mimeToExt {
		extToMime["."+ext] = mime
		if mime == "image/jpeg" {
			extToMime[".jpeg"] = mime
		}
	}

	return &MimeTypeHandler{
		mimeToExt: mimeToExt,
		extToMime: extToMime,
	}
}

func (m *MimeTypeHandler) detectContentType(file io.Reader) (string, io.Reader, error) {
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", nil, err
	}

	contentType := http.DetectContentType(buffer[:n])
	combinedReader := io.MultiReader(bytes.NewReader(buffer[:n]), file)

	ext, ok := m.mimeToExt[contentType]
	if !ok {
		return "", nil, fmt.Errorf("unsupported type: %s", contentType)
	}

	return "." + ext, combinedReader, nil
}

func (m *MimeTypeHandler) getContentType(filename string) string {
	ext := filepath.Ext(filename)
	if mime, ok := m.extToMime[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

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

func randfilename(length int, extension string) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomRunes := make([]rune, length)
	seed := rand.NewSource(time.Now().UnixNano())
	rand := rand.New(seed)
	for index := range randomRunes {
		randomRunes[index] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(randomRunes) + extension
}

func readyzHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "200")
}

func serveImageHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the requested image filename from the URL.
	imageName := filepath.Base(r.URL.Path)

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

func writeFileAndReturnURL(w http.ResponseWriter, r *http.Request, file io.Reader) error {

	hash, err := computeFileHash(file)
	if err != nil {
		return err
	}
	value, exists := imageHashExists(hash)

	if exists {
		if config.Debug {
			fmt.Printf("Hash %s exists: %s\n", hash, value)
		}
		fileURL := constructFileURL(r, value)
		return respondWithFileURL(w, r, fileURL)
	} else {
		if config.Debug {
			fmt.Printf("Hash %s does not exist\n", hash)
		}
		ext, fileReader, err := mimeTypeHandler.detectContentType(file)
		if err != nil {
			http.Error(w, "Unsupported file type", http.StatusBadRequest)
			return err
		}

		genfilename := randfilename(6, ext)
		filepath := filepath.Join(config.UploadPath, genfilename)

		if err := createAndCopyFile(filepath, fileReader); err != nil {
			http.Error(w, "Error processing file", http.StatusInternalServerError)
			return err
		}

		if config.Thumbs {
			if err := makeThumb(filepath); err != nil {
				fmt.Printf("Error creating thumbnail for %s: %v\n", filepath, err)
			}
		}

		hashes[hash] = genfilename

		fileURL := constructFileURL(r, genfilename)
		return respondWithFileURL(w, r, fileURL)
	}
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

func makeThumb(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	img, format, err := image.Decode(file)
	if err != nil {
		return err
	}

	newWidth := img.Bounds().Dx() / 4
	newHeight := img.Bounds().Dy() / 4

	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := x * 4
			srcY := y * 4
			newImg.Set(x, y, img.At(srcX, srcY))
		}
	}

	outputPath := filepath.Join(config.UploadPath, "thumbs", filepath.Base(filename))
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	switch format {
	case "jpeg":
		err = jpeg.Encode(outFile, newImg, nil)
	case "png":
		err = png.Encode(outFile, newImg)
	default:
		err = jpeg.Encode(outFile, newImg, nil)
	}
	if err != nil {
		return err
	}

	// keep timestate stamp
	return os.Chtimes(outputPath, fileInfo.ModTime(), fileInfo.ModTime())
}
