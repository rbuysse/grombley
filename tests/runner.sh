#!/usr/bin/env sh

set -e
trap "docker-compose -f tests/docker-compose.yaml down" EXIT

docker-compose -f tests/docker-compose.yaml build
docker-compose -f tests/docker-compose.yaml up --detach grombley
docker-compose -f tests/docker-compose.yaml up --abort-on-container-exit test-image-upload
