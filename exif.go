package main

import (
	"bytes"
	"fmt"
	"io"

	exif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	jpegstructure "github.com/dsoprea/go-jpeg-image-structure/v2"
	pngstructure "github.com/dsoprea/go-png-image-structure/v2"
)

// isJPEG checks if the data is a JPEG image
func isJPEG(data []byte) bool {
	return len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8
}

// isPNG checks if the data is a PNG image
func isPNG(data []byte) bool {
	return len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47
}

// parseJPEG parses JPEG data and returns a segment list
func parseJPEG(data []byte) (*jpegstructure.SegmentList, error) {
	jmp := jpegstructure.NewJpegMediaParser()
	intfc, err := jmp.ParseBytes(data)
	if err != nil {
		return nil, err
	}
	segments, ok := intfc.(*jpegstructure.SegmentList)
	if !ok {
		return nil, fmt.Errorf("failed to cast to SegmentList")
	}
	return segments, nil
}

// parsePNG parses PNG data and returns a chunk slice
func parsePNG(data []byte) (*pngstructure.ChunkSlice, error) {
	pmp := pngstructure.NewPngMediaParser()
	intfc, err := pmp.ParseBytes(data)
	if err != nil {
		return nil, err
	}
	chunks, ok := intfc.(*pngstructure.ChunkSlice)
	if !ok {
		return nil, fmt.Errorf("failed to cast to ChunkSlice")
	}
	return chunks, nil
}

// buildOrientationExif creates an IfdBuilder with only an orientation tag
func buildOrientationExif(orientation uint16) (*exif.IfdBuilder, error) {
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return nil, err
	}

	ti := exif.NewTagIndex()
	rootIb := exif.NewIfdBuilder(im, ti, exifcommon.IfdStandardIfdIdentity, exifcommon.EncodeDefaultByteOrder)

	err = rootIb.AddStandardWithName("Orientation", []uint16{orientation})
	if err != nil {
		return nil, err
	}

	return rootIb, nil
}

// stripExifButKeepOrientation removes EXIF and metadata from images:
func stripExifButKeepOrientation(data []byte) ([]byte, error) {
	if isJPEG(data) {
		return stripExifFromJpeg(data)
	}

	if isPNG(data) {
		return stripExifFromPng(data)
	}

	// Unknown format, return as-is
	return data, nil
}

func stripExifFromJpeg(data []byte) ([]byte, error) {
	segments, err := parseJPEG(data)
	if err != nil {
		return data, nil
	}

	// Get the orientation value before we strip EXIF
	orientation := getImageOrientation(data)

	// Drop all existing EXIF data
	_, err = segments.DropExif()
	if err != nil {
		return data, nil
	}

	// Create new minimal EXIF with just orientation
	rootIb, err := buildOrientationExif(orientation)
	if err != nil {
		// If we can't create EXIF, just return without EXIF
		var b bytes.Buffer
		segments.Write(&b)
		return b.Bytes(), nil
	}

	// Set the new minimal EXIF
	err = segments.SetExif(rootIb)
	if err != nil {
		// If we can't set EXIF, just return without EXIF
		var b bytes.Buffer
		segments.Write(&b)
		return b.Bytes(), nil
	}

	// Write the modified JPEG
	var b bytes.Buffer
	err = segments.Write(&b)
	if err != nil {
		return data, fmt.Errorf("failed to write modified JPEG: %w", err)
	}

	return b.Bytes(), nil
}

func stripExifFromPng(data []byte) ([]byte, error) {
	chunks, err := parsePNG(data)
	if err != nil {
		return data, nil
	}

	// Get the orientation value and check if EXIF was present
	orientation := getImageOrientation(data)
	_, _, err = chunks.Exif()
	hadExif := (err == nil)

	// Rebuild PNG with minimal EXIF (only orientation if EXIF was present)
	return rebuildPngWithOrientation(chunks, orientation, hadExif)
}

// rebuildPngWithOrientation rebuilds a PNG with minimal EXIF (only orientation tag)
func rebuildPngWithOrientation(chunkSlice *pngstructure.ChunkSlice, orientation uint16, hadExif bool) ([]byte, error) {
	chunks := chunkSlice.Chunks()
	filteredChunks := make([]*pngstructure.Chunk, 0, len(chunks))

	for _, chunk := range chunks {
		// Strip all metadata chunks that might contain sensitive info:
		// - eXIf: EXIF data (GPS, camera info, etc.) - we'll add back minimal version
		// - tEXt: Textual data (can contain arbitrary metadata)
		// - iTXt: International textual data
		// - zTXt: Compressed textual data
		// Keep all other chunks (IHDR, IDAT, PLTE, tRNS, IEND, etc.)
		switch chunk.Type {
		case "eXIf", "tEXt", "iTXt", "zTXt":
			// Skip these metadata chunks
			continue
		default:
			filteredChunks = append(filteredChunks, chunk)
		}
	}

	newChunkSlice := pngstructure.NewChunkSlice(filteredChunks)

	// If there was EXIF data originally, add back minimal EXIF with only orientation
	if hadExif {
		// Create new minimal EXIF with just orientation
		rootIb, err := buildOrientationExif(orientation)
		if err == nil {
			// Set the new minimal EXIF (SetExif takes IfdBuilder directly)
			_ = newChunkSlice.SetExif(rootIb)
			// Ignore errors - if we can't set EXIF, just continue without it
		}
	}

	var b bytes.Buffer
	err := newChunkSlice.WriteTo(&b)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// extractOrientation is a helper to extract orientation from EXIF data
func extractOrientation(exifData []byte) uint16 {
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return 1
	}

	ti := exif.NewTagIndex()
	_, index, err := exif.Collect(im, ti, exifData)
	if err != nil {
		return 1
	}

	results, err := index.RootIfd.FindTagWithName("Orientation")
	if err != nil || len(results) == 0 {
		return 1
	}

	valueRaw, err := results[0].Value()
	if err != nil {
		return 1
	}

	if orientationSlice, ok := valueRaw.([]uint16); ok && len(orientationSlice) > 0 {
		return orientationSlice[0]
	}

	return 1
}

// stripExifButKeepOrientationFromReader is a convenience wrapper
func stripExifButKeepOrientationFromReader(src io.Reader) ([]byte, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	return stripExifButKeepOrientation(data)
}

// getImageOrientation extracts the orientation value from an image (returns 1 if none found)
func getImageOrientation(data []byte) uint16 {
	if isJPEG(data) {
		segments, err := parseJPEG(data)
		if err != nil {
			return 1
		}
		_, exifData, err := segments.Exif()
		if err == nil {
			return extractOrientation(exifData)
		}
	}

	if isPNG(data) {
		chunks, err := parsePNG(data)
		if err != nil {
			return 1
		}
		_, exifData, err := chunks.Exif()
		if err == nil {
			return extractOrientation(exifData)
		}
	}

	return 1
}

// addOrientationTag adds an EXIF orientation tag to an image
func addOrientationTag(data []byte, orientation uint16) ([]byte, error) {
	if isJPEG(data) {
		segments, err := parseJPEG(data)
		if err != nil {
			return data, nil
		}

		// Drop existing EXIF if any
		segments.DropExif()

		// Create minimal EXIF with orientation
		rootIb, err := buildOrientationExif(orientation)
		if err != nil {
			return data, nil
		}
		err = segments.SetExif(rootIb)
		if err != nil {
			return data, nil
		}

		var b bytes.Buffer
		segments.Write(&b)
		return b.Bytes(), nil
	}

	if isPNG(data) {
		chunks, err := parsePNG(data)
		if err != nil {
			return data, nil
		}

		// Create minimal EXIF with orientation
		rootIb, err := buildOrientationExif(orientation)
		if err != nil {
			return data, nil
		}
		_ = chunks.SetExif(rootIb)

		var b bytes.Buffer
		chunks.WriteTo(&b)
		return b.Bytes(), nil
	}

	return data, nil
}
