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
