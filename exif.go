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

// stripExifButKeepOrientation removes EXIF and metadata from images:
// - JPEG: Strips all EXIF data except orientation tag (needed for display)
// - PNG: Strips all EXIF and text metadata chunks completely
func stripExifButKeepOrientation(data []byte) ([]byte, error) {
	// Detect file type
	if len(data) < 2 {
		return data, nil
	}

	// Check if JPEG
	if data[0] == 0xFF && data[1] == 0xD8 {
		return stripExifFromJpeg(data)
	}

	// Check if PNG
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return stripExifFromPng(data)
	}

	// Unknown format, return as-is
	return data, nil
}

func stripExifFromJpeg(data []byte) ([]byte, error) {
	jmp := jpegstructure.NewJpegMediaParser()
	intfc, err := jmp.ParseBytes(data)
	if err != nil {
		return data, nil
	}

	sl, ok := intfc.(*jpegstructure.SegmentList)
	if !ok {
		return data, nil
	}

	// Try to get the orientation value before we strip EXIF
	orientation := uint16(1) // Default orientation
	_, segments, err := sl.Exif()
	if err == nil {
		orientation = extractOrientation(segments)
	}

	// Drop all existing EXIF data
	_, err = sl.DropExif()
	if err != nil {
		return data, nil
	}

	// Create new minimal EXIF with just orientation
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return data, nil
	}

	ti := exif.NewTagIndex()
	rootIb := exif.NewIfdBuilder(im, ti, exifcommon.IfdStandardIfdIdentity, exifcommon.EncodeDefaultByteOrder)

	// Add only the orientation tag
	err = rootIb.AddStandardWithName("Orientation", []uint16{orientation})
	if err != nil {
		// If we can't add orientation, just return without EXIF
		var b bytes.Buffer
		sl.Write(&b)
		return b.Bytes(), nil
	}

	// Set the new minimal EXIF
	err = sl.SetExif(rootIb)
	if err != nil {
		// If we can't set EXIF, just return without EXIF
		var b bytes.Buffer
		sl.Write(&b)
		return b.Bytes(), nil
	}

	// Write the modified JPEG
	var b bytes.Buffer
	err = sl.Write(&b)
	if err != nil {
		return data, fmt.Errorf("failed to write modified JPEG: %w", err)
	}

	return b.Bytes(), nil
}

func stripExifFromPng(data []byte) ([]byte, error) {
	pmp := pngstructure.NewPngMediaParser()
	intfc, err := pmp.ParseBytes(data)
	if err != nil {
		return data, nil
	}

	cs, ok := intfc.(*pngstructure.ChunkSlice)
	if !ok {
		return data, nil
	}

	// For PNG, just remove all EXIF chunks completely
	// PNG orientation handling is inconsistent across browsers anyway
	return rebuildPngWithoutExif(cs)
}

// rebuildPngWithoutExif rebuilds a PNG without any EXIF or metadata chunks
func rebuildPngWithoutExif(cs *pngstructure.ChunkSlice) ([]byte, error) {
	chunks := cs.Chunks()
	filteredChunks := make([]*pngstructure.Chunk, 0, len(chunks))

	for _, chunk := range chunks {
		// Strip all metadata chunks that might contain sensitive info:
		// - eXIf: EXIF data (GPS, camera info, etc.)
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

	newCs := pngstructure.NewChunkSlice(filteredChunks)
	var b bytes.Buffer
	err := newCs.WriteTo(&b)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// extractOrientation is a helper to extract orientation from EXIF segments
func extractOrientation(segments []byte) uint16 {
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return 1
	}

	ti := exif.NewTagIndex()
	_, index, err := exif.Collect(im, ti, segments)
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
