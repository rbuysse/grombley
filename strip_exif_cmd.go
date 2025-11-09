package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runStripExifCommand is the entry point for the strip-exif subcommand
func runStripExifCommand() {
	uploadPath, dryRun, backup, debug := ParseStripExifFlags()

	// Set up a minimal config for debug mode
	config.Debug = debug
	config.UploadPath = uploadPath

	fmt.Printf("Stripping EXIF from images in: %s\n", uploadPath)
	if dryRun {
		fmt.Println("DRY RUN MODE: No files will be modified")
	}
	if backup {
		fmt.Println("BACKUP MODE: Creating .bak files before modification")
	}
	fmt.Println()

	if err := stripExifFromDirectory(uploadPath, dryRun, backup); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// stripExifFromDirectory walks a directory and strips EXIF from all supported images
func stripExifFromDirectory(uploadPath string, dryRun bool, backup bool) error {
	processed := 0
	skipped := 0
	errors := 0

	err := filepath.Walk(uploadPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is a supported image type
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
			return nil
		}

		// Skip GIF files (we don't process them)
		if ext == ".gif" {
			skipped++
			if config.Debug {
				fmt.Printf("Skipping GIF: %s\n", path)
			}
			return nil
		}

		// Read the file
		data, err := os.ReadFile(path)
		if err != nil {
			errors++
			fmt.Printf("Error reading %s: %v\n", path, err)
			return nil // Continue processing other files
		}

		// Strip EXIF but keep orientation
		strippedData, err := stripExifButKeepOrientation(data)
		if err != nil {
			errors++
			fmt.Printf("Error processing %s: %v\n", path, err)
			return nil
		}

		// Check if anything changed
		if len(strippedData) == len(data) {
			skipped++
			if config.Debug {
				fmt.Printf("No change: %s\n", path)
			}
			return nil
		}

		if dryRun {
			fmt.Printf("Would process: %s (size: %d -> %d bytes)\n", path, len(data), len(strippedData))
			processed++
			return nil
		}

		// Create backup if requested
		if backup {
			backupPath := path + ".bak"
			if err := os.WriteFile(backupPath, data, info.Mode()); err != nil {
				errors++
				fmt.Printf("Error creating backup for %s: %v\n", path, err)
				return nil
			}
			if config.Debug {
				fmt.Printf("Created backup: %s\n", backupPath)
			}
		}

		// Write the stripped data back
		if err := os.WriteFile(path, strippedData, info.Mode()); err != nil {
			errors++
			fmt.Printf("Error writing %s: %v\n", path, err)
			return nil
		}

		processed++
		fmt.Printf("Processed: %s (size: %d -> %d bytes)\n", path, len(data), len(strippedData))

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	// Print summary
	fmt.Printf("\nSummary:\n")
	if dryRun {
		fmt.Printf("  Would process: %d files\n", processed)
	} else {
		fmt.Printf("  Processed: %d files\n", processed)
	}
	fmt.Printf("  Skipped: %d files\n", skipped)
	if errors > 0 {
		fmt.Printf("  Errors: %d\n", errors)
	}

	return nil
}
