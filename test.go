package main

import (
	"fmt"

	"github.com/barasher/go-exiftool"
)

func extractRotationFromEXIF(imagePath string) interface {
	et, err := exiftool.NewExiftool()
	if err != nil {
	    fmt.Printf("Error when intializing: %v\n", err)
	    return
	}
	defer et.Close()

	fileInfos := et.ExtractMetadata(imagePath)

	for _, fileInfo := range fileInfos {
	    if fileInfo.Err != nil {
	        fmt.Printf("Error concerning %v: %v\n", fileInfo.File, fileInfo.Err)
	        continue
	    }
	    return fileInfo.Fields["Orientation"]
	    // for k, v := range fileInfo.Fields {
	    //     fmt.Printf("[%v] %v\n", k, v)
	    // }
	}
}

func main() {
	imagePath := "your_image.jpg" // Replace with the path to your image file
	rot := extractRotationFromEXIF(imagePath)
	fmt.Println(rot)
	
}
