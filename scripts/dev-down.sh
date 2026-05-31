#!/usr/bin/env bash
set -e

docker compose --env-file infrastructure/docker/.env -f infrastructure/docker/docker-compose.yml down
