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
  -s, --serve-path     Path to serve images from (default: /i/)
  -u, --upload-path    Path to store uploaded images (default: ./uploads/)`

func GenerateConfig() Config {
	// Parse the command-line flags and load the config
	var bindOpt string
	var configFile string
	var servePathOpt string
	var uploadPathOpt string

	flag.StringVar(&bindOpt, "b", "", "address:port to run the server on")
	flag.StringVar(&bindOpt, "bind", "", "address:port to run the server on")
	flag.StringVar(&configFile, "c", "", "Path to the configuration file")
	flag.StringVar(&configFile, "config", "", "Path to the configuration file")
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

	if configFile == "" {
		configFile = "config.toml"
	}

	// Load the config file if it exists otherwise use default values
	if _, err := os.Stat(configFile); err != nil {
		fmt.Printf("Config file %v not found, using default values\n", configFile)
		config.Bind = "0.0.0.0:3000"
		config.ServePath = "/i/"
		config.UploadPath = "./uploads/"
	} else {
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
