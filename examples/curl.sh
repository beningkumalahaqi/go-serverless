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

body=$(curl -fsS "$BASE_URL/health")
expect health "$body" '"status":"ok"'

body=$(curl -fsS "$BASE_URL/hello?name=Go")
expect hello "$body" '"message":"Hello, Go!"'

body=$(curl -fsS -X POST -H 'Content-Type: application/json' -d '{"message":"ping"}' "$BASE_URL/echo")
expect echo "$body" '"message":"ping"'
expect echo-timestamp "$body" '"receivedAt":"'

status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST -H 'Content-Type: application/json' -d '{}' "$BASE_URL/echo")
[ "$status" = "400" ] || { echo "expected 400, got $status"; exit 1; }
echo "error shape: ok"

curl -fsS "$BASE_URL/openapi.yaml" >/dev/null
curl -fsS "$BASE_URL/docs" >/dev/null
echo "docs: ok"
