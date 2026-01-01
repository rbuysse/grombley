package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const usage = `Usage:
  -b, --bind           address:port to run the server on (default: 0.0.0.0:3000)
  -c, --config         Path to a configuration file (default: config.toml)
  -s, --serve-path     Path to serve images from (default: /i/)
  -u, --upload-path    Path to store uploaded images (default: ./uploads/)`

// Default config
func defaultConfig() Config {
	return Config{
		Bind:       "0.0.0.0:3000",
		ServePath:  "/i/",
		UploadPath: "./uploads/",
	}
}

func GenerateConfig() Config {
	var bindOpt string
	var configFile string
	var configFileSet bool
	var debugOpt bool
	var servePathOpt string
	var uploadPathOpt string

	flag.StringVar(&bindOpt, "b", "", "address:port to run the server on")
	flag.StringVar(&bindOpt, "bind", "", "address:port to run the server on")
	flag.StringVar(&configFile, "c", "", "Path to the configuration file")
	flag.StringVar(&configFile, "config", "", "Path to the configuration file")
	flag.BoolVar(&debugOpt, "debug", false, "enable debug mode")
	flag.StringVar(&servePathOpt, "s", "", "Path to serve images from")
	flag.StringVar(&servePathOpt, "serve-path", "", "Path to serve images from")
	flag.StringVar(&uploadPathOpt, "u", "", "Path to store uploaded images")
	flag.StringVar(&uploadPathOpt, "upload-path", "", "Path to store uploaded images")

	flag.Usage = func() {
		fmt.Println(usage)
	}

	flag.Parse()

	// Check if a config file was specified
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "config" || f.Name == "c" {
			configFileSet = true
		}
	})

	if configFile == "" {
		configFile = "config.toml"
	}

	// Check if the config file exists
	var config Config
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if configFileSet {
			log.Fatalf("Config file %v specified but not found.\n", configFile)
		}
		fmt.Printf("Config file %v not found. Using defaults.\n", configFile)
		config = defaultConfig()
	} else if err != nil {
		log.Fatalf("Error accessing config file %v: %v\n", configFile, err)
	} else {
		// Load the config file
		fmt.Printf("Loading config from %v\n", configFile)
		config = loadConfig(configFile)
	}

	// Override the config values with the command-line flags
	options := map[*string]*string{
		&bindOpt:       &config.Bind,
		&servePathOpt:  &config.ServePath,
		&uploadPathOpt: &config.UploadPath,
	}

	for option, configField := range options {
		if *option != "" {
			*configField = *option
		}
	}

	if debugOpt {
		config.Debug = true
	}

	// Convert upload path to absolute path
	if strings.HasPrefix(config.UploadPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Error getting user home directory: %v\n", err)
		}
		config.UploadPath = filepath.Join(homeDir, config.UploadPath[2:])
	}
	absPath, err := filepath.Abs(config.UploadPath)
	if err != nil {
		log.Fatalf("Error converting upload path to absolute: %v\n", err)
	}
	config.UploadPath = absPath

	return config
}

func loadConfig(configFile string) Config {
	config := defaultConfig()

	// Temporary struct to decode TOML file
	var tempConfig struct {
		Bind       string `toml:"bind"`
		Debug      bool   `toml:"debug"`
		ServePath  string `toml:"serve_path"`
		UploadPath string `toml:"upload_path"`
	}

	if _, err := toml.DecodeFile(configFile, &tempConfig); err != nil {
		log.Fatalf("Error parsing config file %v: %v\n", configFile, err)
	}

	// Merge values from tempConfig into the default config
	if tempConfig.Bind != "" {
		config.Bind = tempConfig.Bind
	}
	if tempConfig.ServePath != "" {
		config.ServePath = tempConfig.ServePath
	}
	if tempConfig.UploadPath != "" {
		config.UploadPath = tempConfig.UploadPath
	}
	if tempConfig.Debug {
		config.Debug = true
	}

	return config
}
