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

`.vercel/` holds generated files after `vercel dev` and `vercel build`. If `gofmt` lists something under `.vercel/`, remove that cache or ignore those paths; do not format generated code.

## 2. Tests

```bash
make test
```

This runs `go vet ./...` and `go test ./... -race -count=1`. The route test imports `api` for its side effects, so every `init` registration is exercised. A failing row prints the status and body that differed.

`TestHandlersGuardMethods` calls each exported handler directly and asserts the 405 response and `Allow` header, which is the behavior a deployed function relies on.

## 3. Portability

```bash
make portability
```

The target fails if `api/`, `core/`, or `docs/` imports a third-party module, or if a Go file outside `api/` mentions Vercel. It prints nothing on success.

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

## 7. Vercel function check

Any change to routing, filenames, or the `api/` package needs the real builder:

```bash
vercel dev --listen 127.0.0.1:18089
```

Then, in another shell:

```bash
BASE_URL=http://127.0.0.1:18089 ./examples/curl.sh
curl -sS -o /dev/null -w 'platform 404: %{http_code}\n' http://127.0.0.1:18089/api/missing
curl -sS -D - -o /dev/null -X POST http://127.0.0.1:18089/api/health | grep -i '^allow'
```

Expected: the script passes, `/api/missing` returns the platform's 404, and the wrong method returns 405 with `Allow: GET, HEAD`. Vercel's build rewrites the module to `handler`, so an import error here often means `api/` reached into `internal/`.

## 8. Production build

Build the function bundles with the real Vercel builder. This catches module, import, and filename problems that `go test` cannot:

```bash
vercel build --yes
find .vercel/output/functions -name '*.func' | sort
```

Every file in `api/` must have a matching `.func` directory. A failure here is the same failure a deployment would hit. Remove `.vercel/output` afterwards if you do not want the artifacts.

## 9. Swagger UI

Open `http://localhost:18080/api/docs` (or the `vercel dev` URL). Confirm:

- All five operations render.
- Try it out shows a request URL under `/api`.
- `/api/health` returns `{"status":"ok"}`.
- `/api/hello?name=Go` returns `{"message":"Hello, Go!"}`.
- `/api/echo` returns the message and a UTC `receivedAt`.

## 10. Deployed check

Only if the change affects routing, deployment config, or handlers, and a deployment exists:

```bash
BASE_URL=https://your-project.vercel.app ./examples/curl.sh
```

Do not report a deployment as verified without this result.

## Failure guide

| Failure | Likely cause |
|---|---|
| `make test` fails with 404 | The test imports no `api` package, the `init` registration is missing, or the path does not match the file name. |
| `make test` fails to compile | Two files export the same handler name, or a file in `api/` exports more than one handler. |
| `make portability` fails | Third-party import in `api/`, `core/`, or `docs/`, or the word Vercel in a Go file outside `api/`. |
| `vercel dev` returns 502 | Build failure, often an `internal/` import from `api/`. Read the `vercel dev` terminal output. |
| `vercel build` fails | A function bundle failed. Check for an `internal/` import, a duplicate handler name, or a file that exports no `http.HandlerFunc`. |
| `vercel build` produces fewer functions than `api/` files | A filename was skipped or the function was not recognized. Confirm `package handler` and one exported handler in that file. |
| Smoke test gets 404 | The server is running old code, or the path lacks `/api`. |
| Smoke test gets 405 | Wrong method for the path. Check the `Allow` header. |
| Redocly fails | Path or response in `docs/openapi.yaml` does not match the files in `api/`. |
| Swagger Try it out fails | Spec URL in `docs/index.html` is wrong, or `servers` is not `/`. |
