#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "health"
curl -fsS "$BASE_URL/health"
echo

echo "hello"
curl -fsS "$BASE_URL/hello?name=Go"
echo

echo "echo"
curl -fsS -X POST -H 'Content-Type: application/json' -d '{"message":"ping"}' "$BASE_URL/echo"
echo

echo "error shape: expects 400"
status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST -H 'Content-Type: application/json' -d '{}' "$BASE_URL/echo")
[ "$status" = "400" ] || { echo "expected 400, got $status"; exit 1; }
echo "400 as expected"

curl -fsS "$BASE_URL/openapi.yaml" >/dev/null
echo "openapi.yaml reachable"

curl -fsS "$BASE_URL/docs" >/dev/null
echo "/docs reachable"
