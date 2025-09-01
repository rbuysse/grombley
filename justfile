all: run

build:
  go build

coverage:
  go test -coverprofile=coverage.out
  go tool cover -html=coverage.out

docker-build:
  docker build -t grombley -f grombley.dockerfile .

docker-run:
  #!/bin/bash
  docker kill grombley &> /dev/null
  docker rm grombley &> /dev/null
  docker run \
    -d \
    --rm \
    --name grombley \
    -p 3000:3000 \
    grombley
  echo -e "Run 'docker kill grombley' to remove the running container."

ci:
  just build
  just fmt

fmt:
  #!/usr/bin/env sh
  if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    gofmt -d -s -l .
    exit 1
  fi
  printf "\033[92mgofmt Success\033[0m\n"

fix-fmt:
  gofmt -w -s .

run *args:
  go run . {{args}}

test:
  go test
  ./tests/runner.sh
