---
name: verify-changes
description: Run the full local verification gate for this Go API before finishing. Use after changing handlers, routes, tests, OpenAPI, docs, or deployment config.
---

# Verify changes

Run every gate. A change is not done while one fails.

## 1. Formatting

```bash
gofmt -l .
```

The command must print nothing. Run `gofmt -w` on any file it lists.

## 2. Tests

```bash
make test
```

This runs `go vet ./...` and `go test ./... -race -count=1`. A failing table row prints the status and body that differed.

## 3. Portability

```bash
make portability
```

The target fails if a core package imports a third-party module or if any Go file mentions Vercel. It prints nothing on success.

## 4. OpenAPI

```bash
npx --yes @redocly/cli@latest lint docs/openapi.yaml
```

The spec must be valid. The current spec produces zero warnings. If a warning appears, fix the spec instead of adding an ignore file.

## 5. Smoke test

```bash
PORT=18080 go run ./cmd/server &
pid=$!
trap 'kill $pid 2>/dev/null || true' EXIT
sleep 2
BASE_URL=http://localhost:18080 ./examples/curl.sh
```

The script asserts response bodies, the echo timestamp, and the 400 error status. It must print `ok` for every check.

## 6. Negative paths

```bash
curl -sS -o /dev/null -w 'root: %{http_code}\n' http://localhost:18080/
curl -sS -o /dev/null -w 'api root: %{http_code}\n' http://localhost:18080/api
curl -sS -o /dev/null -w 'old health: %{http_code}\n' http://localhost:18080/health
curl -sS -o /dev/null -w 'wrong method: %{http_code}\n' -X POST http://localhost:18080/api/health
curl -sS -D - -o /dev/null -X POST http://localhost:18080/api/health | grep -i '^allow'
```

Expected: 404 for `/`, `/api`, and `/health`; 405 for the wrong method; `Allow: GET, HEAD`.

## 7. Swagger UI

Open `http://localhost:18080/api/docs`. Confirm:

- All five operations render.
- Try it out shows a request URL under `/api`.
- `/api/health` returns `{"status":"ok"}`.
- `/api/hello?name=Go` returns `{"message":"Hello, Go!"}`.
- `/api/echo` returns the message and a UTC `receivedAt`.

## 8. Deployed check

Only if the change affects routing, deployment config, or handlers, and a deployment exists:

```bash
BASE_URL=https://your-project.vercel.app ./examples/curl.sh
```

Do not report a deployment as verified without this result.

## Failure guide

| Failure | Likely cause |
|---|---|
| `make test` fails with 404 | Route path and test path disagree, or the `/api` prefix is missing. |
| `make portability` fails | Third-party import in core, or the word Vercel in a Go file. |
| Smoke test gets 404 | The server is running old code, or the path lacks `/api`. |
| Smoke test gets 405 | Wrong method for the path. Check the `Allow` header. |
| Redocly fails | Path or response in `docs/openapi.yaml` does not match the router. |
| Swagger Try it out fails | Spec URL in `docs/index.html` is wrong, or `servers` is not `/`. |
