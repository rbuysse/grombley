package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// Helper function to calculate expected absolute path from a relative path
func expectedAbsPath(t *testing.T, relPath string) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	absPath, err := filepath.Abs(filepath.Join(cwd, relPath))
	if err != nil {
		t.Fatalf("Error calculating expected path: %v", err)
	}
	return absPath
}

func TestConfig(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		config := defaultConfig()

		if config.Bind != "0.0.0.0:3000" {
			t.Errorf("Expected default bind to be 0.0.0.0:3000, but got %s", config.Bind)
		}

		if config.ServePath != "/i/" {
			t.Errorf("Expected default serve_path to be /i/, but got %s", config.ServePath)
		}

		if config.UploadPath != "./uploads/" {
			t.Errorf("Expected default upload_path to be ./uploads/, but got %s", config.UploadPath)
		}

		if config.Debug {
			t.Errorf("Expected default debug to be false, but got true")
		}
	})

	t.Run("load full config from file", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "config-*.toml")
		if err != nil {
			t.Fatalf("Error creating temporary file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		configContent := `
bind = "localhost:666"
serve_path = "/p/"
upload_path = "./grapes/"
`
		if _, err := tempFile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Error writing to temporary file: %v", err)
		}

		config := loadConfig(tempFile.Name())

		if config.Bind != "localhost:666" {
			t.Errorf("Expected bind to be localhost:666, but got %s", config.Bind)
		}

		if config.ServePath != "/p/" {
			t.Errorf("Expected serve_path to be /p/, but got %s", config.ServePath)
		}

		if config.UploadPath != "./grapes/" {
			t.Errorf("Expected upload_path to be ./grapes/, but got %s", config.UploadPath)
		}

		if config.Debug {
			t.Errorf("Expected debug to be false, but got true")
		}
	})

	t.Run("load partial config with defaults", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "config-partial-*.toml")
		if err != nil {
			t.Fatalf("Error creating temporary file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		partialConfigContent := `bind = "localhost:777"`
		if _, err := tempFile.Write([]byte(partialConfigContent)); err != nil {
			t.Fatalf("Error writing to temporary file: %v", err)
		}

		config := loadConfig(tempFile.Name())

		if config.Bind != "localhost:777" {
			t.Errorf("Expected bind to be localhost:777, but got %s", config.Bind)
		}

		// Should use defaults for missing values
		if config.ServePath != "/i/" {
			t.Errorf("Expected serve_path to be /i/ (default), but got %s", config.ServePath)
		}

		if config.UploadPath != "./uploads/" {
			t.Errorf("Expected upload_path to be ./uploads/ (default), but got %s", config.UploadPath)
		}
	})

	t.Run("load debug flag from config file", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "config-debug-*.toml")
		if err != nil {
			t.Fatalf("Error creating temporary file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		configContent := `
bind = "localhost:8080"
debug = true
`
		if _, err := tempFile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Error writing to temporary file: %v", err)
		}

		config := loadConfig(tempFile.Name())

		if !config.Debug {
			t.Errorf("Expected debug to be true, but got false")
		}

		if config.Bind != "localhost:8080" {
			t.Errorf("Expected bind to be localhost:8080, but got %s", config.Bind)
		}
	})

	t.Run("load debug false from config file", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "config-no-debug-*.toml")
		if err != nil {
			t.Fatalf("Error creating temporary file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		configContent := `
bind = "localhost:8080"
debug = false
`
		if _, err := tempFile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Error writing to temporary file: %v", err)
		}

		config := loadConfig(tempFile.Name())

		if config.Debug {
			t.Errorf("Expected debug to be false, but got true")
		}
	})

	t.Run("cli flag debug without config file", func(t *testing.T) {
		// Reset flag.CommandLine for each test to avoid conflicts
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "--debug"}

		config := GenerateConfig()

		if !config.Debug {
			t.Errorf("Expected debug to be true when --debug flag is provided, but got false")
		}

		// Should still have default values
		if config.Bind != "0.0.0.0:3000" {
			t.Errorf("Expected default bind, but got %s", config.Bind)
		}
	})

	t.Run("cli flag debug overrides config file", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		tempFile, err := os.CreateTemp("", "config-test-*.toml")
		if err != nil {
			t.Fatalf("Error creating temporary file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		configContent := `
bind = "localhost:9000"
debug = false
`
		if _, err := tempFile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Error writing to temporary file: %v", err)
		}

		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "--config", tempFile.Name(), "--debug"}

		config := GenerateConfig()

		// Command line flag should override config file
		if !config.Debug {
			t.Errorf("Expected debug to be true (CLI flag overrides config file), but got false")
		}

		if config.Bind != "localhost:9000" {
			t.Errorf("Expected bind from config file to be localhost:9000, but got %s", config.Bind)
		}
	})

	t.Run("cli flags override all config file values", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		tempFile, err := os.CreateTemp("", "config-override-*.toml")
		if err != nil {
			t.Fatalf("Error creating temporary file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		configContent := `
bind = "localhost:9000"
serve_path = "/images/"
upload_path = "./files/"
debug = false
`
		if _, err := tempFile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Error writing to temporary file: %v", err)
		}

		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd",
			"--config", tempFile.Name(),
			"--bind", "127.0.0.1:8888",
			"--serve-path", "/s/",
			"--upload-path", "./custom/",
			"--debug",
		}

		config := GenerateConfig()

		// All values should come from command line flags
		if config.Bind != "127.0.0.1:8888" {
			t.Errorf("Expected bind from CLI flag to be 127.0.0.1:8888, but got %s", config.Bind)
		}

		if config.ServePath != "/s/" {
			t.Errorf("Expected serve_path from CLI flag to be /s/, but got %s", config.ServePath)
		}

		// Upload path should be converted to absolute
		expectedUploadPath := expectedAbsPath(t, "./custom/")
		if config.UploadPath != expectedUploadPath {
			t.Errorf("Expected upload_path to be %s, but got %s", expectedUploadPath, config.UploadPath)
		}

		if !config.Debug {
			t.Errorf("Expected debug from CLI flag to be true, but got false")
		}
	})

	t.Run("cli short flags work", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd",
			"-b", "localhost:7777",
			"-s", "/media/",
			"-u", "./temp/",
			"--debug",
		}

		config := GenerateConfig()

		if config.Bind != "localhost:7777" {
			t.Errorf("Expected bind from -b flag to be localhost:7777, but got %s", config.Bind)
		}

		if config.ServePath != "/media/" {
			t.Errorf("Expected serve_path from -s flag to be /media/, but got %s", config.ServePath)
		}

		// Upload path should be converted to absolute
		expectedUploadPath := expectedAbsPath(t, "./temp/")
		if config.UploadPath != expectedUploadPath {
			t.Errorf("Expected upload_path to be %s, but got %s", expectedUploadPath, config.UploadPath)
		}

		if !config.Debug {
			t.Errorf("Expected debug to be true, but got false")
		}
	})

	t.Run("expand tilde in upload path", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Error getting user home directory: %v", err)
		}

		tempFile, err := os.CreateTemp("", "config-*.toml")
		if err != nil {
			t.Fatalf("Error creating temporary file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		configContent := `
bind = "localhost:3000"
upload_path = "~/test/uploads"
`
		if _, err := tempFile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Error writing to temporary file: %v", err)
		}
		tempFile.Close()

		// Reset flags for this test
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-c", tempFile.Name()}

		config := GenerateConfig()

		expectedUploadPath := homeDir + "/test/uploads"
		if config.UploadPath != expectedUploadPath {
			t.Errorf("Expected upload_path to be %s, but got %s", expectedUploadPath, config.UploadPath)
		}
	})

	t.Run("convert relative paths to absolute", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "config-*.toml")
		if err != nil {
			t.Fatalf("Error creating temporary file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		configContent := `
bind = "localhost:3000"
upload_path = "../../grims"
`
		if _, err := tempFile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Error writing to temporary file: %v", err)
		}
		tempFile.Close()

		// Reset flags for this test
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-c", tempFile.Name()}

		config := GenerateConfig()

		// Calculate what the expected absolute path should be
		// ../../grims from the current working directory
		expectedUploadPath := expectedAbsPath(t, "../../grims")

		if config.UploadPath != expectedUploadPath {
			t.Errorf("Expected upload_path to be %s, but got %s", expectedUploadPath, config.UploadPath)
		}
	})
}
