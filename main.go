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
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

type Gallery struct {
	ID      string    `json:"id"`
	Images  []string  `json:"images"`
	Created time.Time `json:"created"`
}

type Config struct {
	Bind       string `toml:"bind"`
	Debug      bool   `toml:"debug"`
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

	// create galleries directory if it doesn't exist
	if _, err := os.Stat("galleries"); os.IsNotExist(err) {
		os.MkdirAll("galleries", os.ModePerm)
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
	http.HandleFunc("/g/", galleryHandler)
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

// Serve thumbnail (1/4 size)
func serveThumbnailImageHandler(w http.ResponseWriter, r *http.Request) {
	imageName := filepath.Base(r.URL.Path)
	if err := validateImageName(imageName, config.UploadPath); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	imagePath := filepath.Join(config.UploadPath, imageName)
	imageFile, err := os.Open(imagePath)
	if err != nil {
		notfoundHandler(w)
		return
	}
	defer imageFile.Close()

	dst, format, err := shrinkImage(imageFile, 4)
	if err != nil {
		http.Error(w, "Failed to shrink image", http.StatusInternalServerError)
		return
	}

	if format == "png" {
		w.Header().Set("Content-Type", "image/png")
		err = png.Encode(w, dst)
	} else {
		w.Header().Set("Content-Type", "image/jpeg")
		err = jpeg.Encode(w, dst, &jpeg.Options{Quality: 85})
	}
	if err != nil {
		// Can't send error after starting to write response body
		log.Printf("Failed to encode thumbnail image: %v", err)
		return
	}
}

// getExifOrientation reads EXIF orientation from image data
func getExifOrientation(reader io.Reader) int {
	x, err := exif.Decode(reader)
	if err != nil {
		return 1 // default orientation if no EXIF data
	}

	orientation, err := x.Get(exif.Orientation)
	if err != nil {
		return 1 // default orientation if no orientation tag
	}

	orientationVal, err := orientation.Int(0)
	if err != nil {
		return 1
	}

	return orientationVal
}

// applyOrientation applies EXIF orientation transformations to an image
func applyOrientation(img image.Image, orientation int) image.Image {
	if orientation == 1 {
		return img // no transformation needed
	}

	bounds := img.Bounds()
	var dst *image.RGBA

	switch orientation {
	case 2: // flip horizontal
		dst = image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.Set(bounds.Max.X-1-x+bounds.Min.X, y, img.At(x, y))
			}
		}
	case 3: // rotate 180
		dst = image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.Set(bounds.Max.X-1-x+bounds.Min.X, bounds.Max.Y-1-y+bounds.Min.Y, img.At(x, y))
			}
		}
	case 4: // flip vertical
		dst = image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.Set(x, bounds.Max.Y-1-y+bounds.Min.Y, img.At(x, y))
			}
		}
	case 5: // transpose (flip horizontal and rotate 270 CW)
		dst = image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.Set(y-bounds.Min.Y, x-bounds.Min.X, img.At(x, y))
			}
		}
	case 6: // rotate 90 CW
		dst = image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.Set(bounds.Max.Y-1-y+bounds.Min.Y, x-bounds.Min.X, img.At(x, y))
			}
		}
	case 7: // transverse (flip horizontal and rotate 90 CW)
		dst = image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.Set(bounds.Max.Y-1-y+bounds.Min.Y, bounds.Max.X-1-x+bounds.Min.X, img.At(x, y))
			}
		}
	case 8: // rotate 270 CW
		dst = image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.Set(y-bounds.Min.Y, bounds.Max.X-1-x+bounds.Min.X, img.At(x, y))
			}
		}
	default:
		return img
	}

	return dst
}

// shrinkImage reduces the size of an image by the given factor and returns the new image and format.
func shrinkImage(reader io.Reader, factor int) (image.Image, string, error) {
	// Read all data first to allow multiple reads
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}

	// Try to decode EXIF orientation
	orientation := getExifOrientation(bytes.NewReader(data))

	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}

	// Apply EXIF orientation transformations
	img = applyOrientation(img, orientation)

	bounds := img.Bounds()
	newW := bounds.Dx() / factor
	newH := bounds.Dy() / factor
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := x * factor
			srcY := y * factor
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}
	return dst, format, nil
}

// processUploadedFile handles a single file upload and returns the filename
func processUploadedFile(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash, err := computeFileHash(file)
	if err != nil {
		return "", err
	}

	// check if file already exists
	if value, exists := imageHashExists(hash); exists {
		return value, nil
	}

	// reset file reader after hash computation
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, 0)
	} else {
		file.Close()
		file, err = fileHeader.Open()
		if err != nil {
			return "", err
		}
		defer file.Close()
	}

	ext, fileReader, err := mimeTypeHandler.detectContentType(file)
	if err != nil {
		return "", err
	}

	genfilename := randfilename(6, ext)
	filepath := filepath.Join(config.UploadPath, genfilename)

	if err := createAndCopyFile(filepath, fileReader); err != nil {
		return "", err
	}

	hashes[hash] = genfilename
	return genfilename, nil
}

// createGallery saves gallery data and returns the gallery URL
func createGallery(r *http.Request, imageFilenames []string) (string, error) {
	galleryID := randfilename(6, "")
	gallery := Gallery{
		ID:      galleryID,
		Images:  imageFilenames,
		Created: time.Now(),
	}

	galleryPath := filepath.Join("galleries", galleryID+".json")
	os.MkdirAll("galleries", os.ModePerm)

	galleryJSON, err := json.Marshal(gallery)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(galleryPath, galleryJSON, 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf("http://%s/g/%s", r.Host, galleryID), nil
}

func galleryHandler(w http.ResponseWriter, r *http.Request) {
	galleryID := strings.TrimPrefix(r.URL.Path, "/g/")
	if galleryID == "" {
		notfoundHandler(w)
		return
	}

	galleryPath := filepath.Join("galleries", galleryID+".json")
	galleryData, err := os.ReadFile(galleryPath)
	if err != nil {
		notfoundHandler(w)
		return
	}

	var gallery Gallery
	if err := json.Unmarshal(galleryData, &gallery); err != nil {
		notfoundHandler(w)
		return
	}

	// build both thumbnail and full urls for images
	type ImagePair struct {
		Thumbnail string
		Full      string
	}
	var imagePairs []ImagePair
	for _, img := range gallery.Images {
		imagePairs = append(imagePairs, ImagePair{
			Thumbnail: fmt.Sprintf("/t/%s", img),
			Full:      fmt.Sprintf("%s%s", config.ServePath, img),
		})
	}

	// serve gallery template
	tmpl, err := template.ParseFS(templatesFolder, "templates/gallery.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, imagePairs)
}
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB max in-memory size
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// get all uploaded files
	var files []*multipart.FileHeader
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		files = r.MultipartForm.File["file"]
	}

	// fallback to single file method if no files from multipart
	if len(files) == 0 {
		file, _, err := r.FormFile("file")
		if err != nil {
			fmt.Println("Error retrieving the file:", err)
			http.Error(w, "Error retrieving the file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		writeFileAndReturnURL(w, r, file)
		return
	}

	// single file - return direct URL
	if len(files) == 1 {
		file, err := files[0].Open()
		if err != nil {
			http.Error(w, "Error opening file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		writeFileAndReturnURL(w, r, file)
		return
	}

	// multiple files - process and create gallery
	var imageFilenames []string
	for _, fileHeader := range files {
		filename, err := processUploadedFile(fileHeader)
		if err != nil {
			continue // skip files that fail to process
		}
		imageFilenames = append(imageFilenames, filename)
	}

	if len(imageFilenames) == 0 {
		http.Error(w, "No valid images uploaded", http.StatusBadRequest)
		return
	}

	// create and return gallery URL
	galleryURL, err := createGallery(r, imageFilenames)
	if err != nil {
		http.Error(w, "Error creating gallery", http.StatusInternalServerError)
		return
	}

	respondWithFileURL(w, r, galleryURL)
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

// validateImageName checks for path traversal, empty names, and allowed extensions
func validateImageName(imageName string, uploadPath string) error {
	if imageName == "" || imageName == "." || imageName == ".." {
		return fmt.Errorf("invalid file name")
	}

	if strings.HasPrefix(imageName, ".") {
		return fmt.Errorf("invalid file name")
	}

	imagePath := filepath.Join(uploadPath, imageName)
	absImagePath, err := filepath.Abs(imagePath)
	absUploadPath, err2 := filepath.Abs(uploadPath)
	if err != nil || err2 != nil || !strings.HasPrefix(absImagePath, absUploadPath) {
		return fmt.Errorf("invalid file path")
	}

	ext := strings.ToLower(filepath.Ext(imageName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		return nil
	default:
		return fmt.Errorf("unsupported file type")
	}
}
