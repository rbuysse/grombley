package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

const usage = `Usage:
  -c, --config         Path to a configuration file (default: config.toml)
  -p, --port           Port to run the server on (default: 3000)
  -s, --serve-path     Path to serve images from (default: /i/)
  -u, --upload-path    Path to store uploaded images (default: ./uploads/)`

func GenerateConfig() Config {
	// Parse the command-line flags and load the config
	var configFile string
	var portOpt string
	var servePathOpt string
	var uploadPathOpt string

	flag.StringVar(&configFile, "c", "config.toml", "Path to the configuration file")
	flag.StringVar(&configFile, "config", "config.toml", "Path to the configuration file")
	flag.StringVar(&portOpt, "p", "3000", "Port to run the server on")
	flag.StringVar(&portOpt, "port", "3000", "Port to run the server on")
	flag.StringVar(&servePathOpt, "s", "", "Path to serve images from")
	flag.StringVar(&servePathOpt, "serve-path", "", "Path to serve images from")
	flag.StringVar(&uploadPathOpt, "u", "", "Path to store uploaded images")
	flag.StringVar(&uploadPathOpt, "upload-path", "", "Path to store uploaded images")

	flag.Usage = func() {
		fmt.Println(usage)
	}

	if !flag.Parsed() {
		flag.Parse()
	}

	// Load the config file if it exists otherwise use default values
	if _, err := os.Stat(configFile); err != nil {
		config.Port = "3000"
		config.ServePath = "/i/"
		config.UploadPath = "./uploads/"
	} else {
		config = loadConfig(configFile)
	}

	// Override the config values with the command-line flags
	options := map[*string]*string{
		&portOpt:       &config.Port,
		&servePathOpt:  &config.ServePath,
		&uploadPathOpt: &config.UploadPath,
	}

	for option, configField := range options {
		if *option != "" {
			*configField = *option
		}
	}

	return config
}

func loadConfig(configFile string) Config {

	var config Config

	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Fatalf("Error in parsing config file: %v", err)
		os.Exit(1)
	}

	return config
}
