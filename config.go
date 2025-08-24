package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

const usage = `Usage:
  -b, --bind           address:port to run the server on (default: 0.0.0.0:3000)
  -c, --config         Path to a configuration file (default: config.toml)
  -g, --gallery-path   Path to store gallery metadata (default: ./galleries/)
  -s, --serve-path     Path to serve images from (default: /i/)
  -u, --upload-path    Path to store uploaded images (default: ./uploads/)`

// Default config
func defaultConfig() Config {
	return Config{
		Bind:        "0.0.0.0:3000",
		ServePath:   "/i/",
		UploadPath:  "./uploads/",
		GalleryPath: "./galleries/",
	}
}

func GenerateConfig() Config {
	var bindOpt string
	var configFile string
	var configFileSet bool
	var debugOpt bool
	var servePathOpt string
	var uploadPathOpt string
	var galleryPathOpt string

	flag.StringVar(&bindOpt, "b", "", "address:port to run the server on")
	flag.StringVar(&bindOpt, "bind", "", "address:port to run the server on")
	flag.StringVar(&configFile, "c", "", "Path to the configuration file")
	flag.StringVar(&configFile, "config", "", "Path to the configuration file")
	flag.BoolVar(&debugOpt, "debug", false, "enable debug mode")
	flag.StringVar(&galleryPathOpt, "g", "", "Path to store gallery metadata")
	flag.StringVar(&galleryPathOpt, "gallery-path", "", "Path to store gallery metadata")
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
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if configFileSet {
			log.Fatalf("Config file %v specified but not found.\n", configFile)
		}
		fmt.Printf("Config file %v not found. Using defaults.\n", configFile)
		return defaultConfig()
	} else if err != nil {
		log.Fatalf("Error accessing config file %v: %v\n", configFile, err)
	}

	// Load the config file
	fmt.Printf("Loading config from %v\n", configFile)
	config := loadConfig(configFile)

	// Override the config values with the command-line flags
	options := map[*string]*string{
		&bindOpt:        &config.Bind,
		&galleryPathOpt: &config.GalleryPath,
		&servePathOpt:   &config.ServePath,
		&uploadPathOpt:  &config.UploadPath,
	}

	for option, configField := range options {
		if *option != "" {
			*configField = *option
		}
	}

	if debugOpt {
		config.Debug = true
	}

	return config
}

func loadConfig(configFile string) Config {
	config := defaultConfig()

	// Temporary struct to decode TOML file
	var tempConfig struct {
		Bind        string `toml:"bind"`
		ServePath   string `toml:"serve_path"`
		UploadPath  string `toml:"upload_path"`
		GalleryPath string `toml:"gallery_path"`
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
	if tempConfig.GalleryPath != "" {
		config.GalleryPath = tempConfig.GalleryPath
	}

	return config
}
