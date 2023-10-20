package main

import (
	"fmt"

	"github.com/barasher/go-exiftool"
)

func main() {

	et, err := exiftool.NewExiftool()
	if err != nil {
	    fmt.Printf("Error when intializing: %v\n", err)
	    return
	}
	defer et.Close()

	fileInfos := et.ExtractMetadata("your_image.jpg")

	for _, fileInfo := range fileInfos {
	    if fileInfo.Err != nil {
	        fmt.Printf("Error concerning %v: %v\n", fileInfo.File, fileInfo.Err)
	        continue
	    }

	    for k, v := range fileInfo.Fields {
	        fmt.Printf("[%v] %v\n", k, v)
	    }
	}
}
