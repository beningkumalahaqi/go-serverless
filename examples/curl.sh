#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

expect() {
  local name="$1" body="$2" needle="$3"
  case "$body" in
    *"$needle"*) printf '%s: ok\n' "$name" ;;
    *)
      printf '%s: unexpected body: %s\n' "$name" "$body" >&2
      exit 1
      ;;
  esac
}

body=$(curl -fsS "$BASE_URL/api/health")
expect health "$body" '"data":{"status":"ok"}'

body=$(curl -fsS "$BASE_URL/api/hello?name=Go")
expect hello "$body" '"data":{"message":"Hello, Go!"}'

body=$(curl -fsS -X POST -H 'Content-Type: application/json' -d '{"message":"ping"}' "$BASE_URL/api/echo")
expect echo "$body" '"data":{"message":"ping"'
expect echo-timestamp "$body" '"receivedAt":"'

response=$(curl -sS -w '\n%{http_code}' -X POST -H 'Content-Type: application/json' -d '{}' "$BASE_URL/api/echo")
body=${response%$'\n'*}
status=${response##*$'\n'}
[ "$status" = "400" ] || { echo "expected 400, got $status"; exit 1; }
expect error-code "$body" '"code":"invalid_request"'
expect error-shape "$body" '"error":{'
echo "error shape: ok"

curl -fsS "$BASE_URL/api/openapi.yaml" >/dev/null
curl -fsS "$BASE_URL/api/docs" >/dev/null
echo "docs: ok"
