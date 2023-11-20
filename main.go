package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	// "github.com/disintegration/gift"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/scottleedavis/go-exif-remove"
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

	// Define the port you want the server to listen on
	port := config.Port // Change this to your desired port

	fmt.Printf("Server is running on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

// func fixOrientation (tmppath string) {
// 	src := loadImage(tmppath)

// 	filters := map[string]gift.Filter{
// 	}
// }

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

func getOrientation(file io.Reader) int {
	orientationMap := make(map[int]int)
	// I think 6 and 8 are swapped even though that disagrees with
	// https://jdhao.github.io/2019/07/31/image_rotation_exif_info/
	orientationMap[1] = 0
	orientationMap[8] = 90
	orientationMap[3] = 180
	orientationMap[6] = 270

	// f, err := os.Open(filepath) // delete this I guess
	x, err := exif.Decode(file)
	if err != nil {
		return 0
	}

	orientationTag, err := x.Get(exif.Orientation)
	if err != nil {
		log.Printf("Error getting orientation: %v", err)
		return 0
	}

	orientationInt, err := orientationTag.Int(0)
	if err != nil {
		log.Printf("Error converting orientation to int: %v", err)
		return 0
	}
	fmt.Println("-- DEBUG", orientationInt)
	return orientationMap[orientationInt]
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templatesFolder, "templates/index.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl.Execute(w, nil)
}

// func loadImage(filename string) image.Image {
// 	f, err := os.Open(filename)
// 	if err != nil {
// 		log.Fatalf("os.Open failed: %v", err)
// 	}
// 	defer f.Close()
// 	img, _, err := image.Decode(f)
// 	if err != nil {
// 		log.Fatalf("image.Decode failed: %v", err)
// 	}
// 	return img
// }

func randfilename(n int, f string) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	extension := strings.Split(f, ".")[1]
	return string(b) + "." + extension
}

func removeEXIF(input io.Reader) (io.Reader, error) {
	// Read the input data into a []byte
	inputData, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	// Remove EXIF data
	processedData, err := exifremove.Remove(inputData)
	if err != nil {
		return nil, err
	}

	// Create a new io.Reader from the processed data
	output := bytes.NewReader(processedData)

	return output, nil
}

// func saveImage(filename string, img image.Image) {
// 	f, err := os.Create(filename)
// 	if err != nil {
// 		log.Fatalf("os.Create failed: %v", err)
// 	}
// 	defer f.Close()
// 	err = png.Encode(f, img)
// 	if err != nil {
// 		log.Fatalf("png.Encode failed: %v", err)
// 	}
// }

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

	parts := strings.Split(parsedUrl.Path, "/")
	filename := parts[len(parts)-1]

	writeFileAndRedirect(w, r, resp.Body, filename)
}

func writeFileAndRedirect(w http.ResponseWriter, r *http.Request, file io.Reader, filename string) {
	filename = strings.ToLower(filename)
	genfilename := randfilename(6, filename)
	tmpFile, err := os.Create("/tmp/" + genfilename)
	if err != nil {
		fmt.Println("Error creating the file:", err)
		http.Error(w, "Error creating the file", http.StatusInternalServerError)
		return
	}
	defer tmpFile.Close()

	// io.Reader is readonce so we have to make a copy to read for exif
	// and read for writing the file
	var buf bytes.Buffer
	tee := io.TeeReader(file, &buf)

	a, _ := ioutil.ReadAll(tee)
	b, _ := ioutil.ReadAll(&buf)

	// grab the exif orientation value
	o := getOrientation(bytes.NewReader(b))
	fmt.Println(o)

	// remove a bunch of exif
	fileNoExif, err := removeEXIF(bytes.NewReader(a))

	_, err = io.Copy(tmpFile, fileNoExif)
	if err != nil {
		fmt.Println("Error copying file data:", err)
		http.Error(w, "Error copying file data", http.StatusInternalServerError)
		return
	}

	// write file out to a temp dir
	renameErr := os.Rename("/tmp/"+genfilename, config.UploadPath+genfilename)

	if err != nil {
		fmt.Println(renameErr)
		return
	}

	// http.Redirect(w, r, "/i/"+genfilename, http.StatusSeeOther)
}
