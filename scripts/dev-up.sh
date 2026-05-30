#!/usr/bin/env bash
set -e

docker compose -f infrastructure/docker/docker-compose.yml up --build -d