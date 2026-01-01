# Grombley

grombley is a self-hosted image hosting service with very few features.

First,
1. Install [just](https://github.com/casey/just)
1. Clone the grombley repo, `cd grombley`

Then,

## Build locally
  1. Install [Golang](https://go.dev/doc/install)
  1. Run `just build`
  1. Open [http://localhost:3000](http://localhost:3000)
     to use grombley

## Docker
  1. Run `just docker-build`
  1. Run `just docker-run` 
  1. Open [http://localhost:3000](http://localhost:3000)
     to use grombley

## Configuration Options

You can configure the application using a TOML file (default: `config.toml`)
or command-line flags. An example config file is provided at
`config.toml.example`.

### Available Options

Command-line flags override values in the config file. If no config file is
found, defaults are used.

| Option         | TOML Key     | CLI Flag(s)                | Default Value      | Description                           |
|----------------|--------------|----------------------------|--------------------|---------------------------------------|
| Config file    | —            | `-c`, `--config`           | `config.toml`      | Path to the configuration file        |
| Bind address   | `bind`       | `-b`, `--bind`             | `0.0.0.0:3000`     | Address and port to run the server on |
| Debug mode     | `debug`      | `--debug`                  | `false`            | Enable debug mode                     |
| Serve path     | `serve_path` | `-s`, `--serve-path`       | `/i/`              | Path to serve images from             |
| Upload path    | `upload_path`| `-u`, `--upload-path`      | `./uploads/`       | Path to store uploaded images         |
