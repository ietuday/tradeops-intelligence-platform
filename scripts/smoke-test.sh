#!/usr/bin/env bash
set -e

API_URL="${API_URL:-http://localhost:8080}"

echo "Running smoke tests against ${API_URL}"

echo "Checking /health..."
curl -fsS "${API_URL}/health" | grep "api-gateway"

echo "Checking /ready..."
curl -fsS "${API_URL}/ready" | grep "api-gateway"

echo "Checking /metrics..."
curl -fsS "${API_URL}/metrics" | grep "process_cpu"

echo "Smoke tests passed."