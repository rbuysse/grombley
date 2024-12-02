package main

import (
	"bytes"
	"image/jpeg"
	"io"
	"log"

	"github.com/jdeng/goheif"
)

func isHEIC(header []byte) bool {
	// heic files start with a 'ftyp' box after a 4-byte size.
	// [4 bytes size][4 bytes 'ftyp'][brand identifier]
	// https://mp4ra.org/registered-types/brands
	signatures := [][]byte{
		[]byte("ftypheic"),
	}

	for _, sig := range signatures {
		if len(header) >= len(sig)+4 {
			if bytes.Contains(header[4:], sig[4:]) {
				return true
			}
		}
	}
	return false
}

func convertHEICToJPEG(src io.Reader) (io.Reader, error) {
	img, err := goheif.Decode(src)
	if err != nil {
		log.Printf("HEIC decoding failed: %v", err)
		return nil, err
	}
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 90}); err != nil {
		log.Printf("JPEG encoding failed: %v", err)
		return nil, err
	}
	return buf, nil
}
