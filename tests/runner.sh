#!/usr/bin/env sh

set -e
trap "docker compose -f tests/docker-compose.yaml down > /dev/null 2>&1" EXIT

if [ "$(docker ps -q -f name=grombley)" ]; then
  docker kill grombley
fi

docker compose -f tests/docker-compose.yaml build
docker compose -f tests/docker-compose.yaml up --detach grombley
docker compose -f tests/docker-compose.yaml up --detach nginx
docker compose -f tests/docker-compose.yaml up --abort-on-container-exit test

docker compose -f tests/docker-compose.yaml down
