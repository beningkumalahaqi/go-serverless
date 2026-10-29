# Go Serverless API

A small Go HTTP API that deploys to Vercel as one function per file and runs unchanged on any host that can execute a binary.

- Three example endpoints, a shared JSON error shape, and tests.
- An OpenAPI 3.0 spec and Swagger UI served by the app itself.
- One Go file per endpoint, plus a local server that serves the same routes.

The code uses the Go standard library only. There is no code generation and no framework.

Scope note: auth, database, queues, caching, and secrets management are intentionally absent. Add them when the project needs them.

> The deployed API has no authentication. Every endpoint is public. Add auth or a rate limit before you put real data behind it.

## Prerequisites

| Tool | Needed for |
|---|---|
| Go 1.26 or newer | Running and testing locally |
| `make` | Short commands below (optional, every target maps to a `go` command) |
| Node.js and npm | Installing the Vercel CLI |
| A Vercel account | Deploying |

## Run locally

```bash
git clone <this-repo> go-serverless
cd go-serverless
make run
```

Open <http://localhost:8080/api/docs> for Swagger UI. The same calls without a browser:

```bash
curl localhost:8080/api/health
curl 'localhost:8080/api/hello?name=Go'
curl -X POST localhost:8080/api/echo \
  -H 'Content-Type: application/json' \
  -d '{"message":"ping"}'
```

`make run` is plain `go run ./cmd/server`.

Stop the server with Ctrl+C.

## Test

```bash
make test        # go vet + go test -race
make portability # shared packages stay stdlib-only; only api/ may mention Vercel
```

Both commands must pass before a change is done.

## Endpoints

Every route is under `/api`. Each URL comes from a file in `api/`: `api/health.go` becomes `/api/health`, and `api/openapi.yaml.go` becomes `/api/openapi.yaml`.

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/health` | Liveness check |
| GET | `/api/hello?name=World` | Greeting, `name` optional |
| POST | `/api/echo` | Echo a JSON `message` (max 1 KB) with a UTC timestamp |
| GET | `/api/docs` | Swagger UI |
| GET | `/api/openapi.yaml` | OpenAPI 3.0.3 document |

Errors use one shape:

```json
{ "error": { "code": "invalid_request", "message": "message is required" } }
```

Status codes: `200`, `400` for bad input, `404` for unknown routes, `405` with an `Allow` header for wrong methods, `413` for a body over 1 KB. Under `cmd/server`, unknown paths including `/` and `/api` return the JSON 404 shape. On Vercel, an unknown path never reaches a function, so the platform's HTML 404 is returned instead. Open `/api/docs` for the UI.

## Deploy to Vercel

`vercel.json` sets `"framework": null`. Vercel's zero-config Go builder turns every `.go` file in `api/` into its own function and maps the file name to the URL. There is no build command and no central route table.

Adding an endpoint means adding one file. Redeploy and the new URL exists. Local `cmd/server` picks the file up on the next build because every file registers itself during `init()`.

Vercel's Go runtime is in beta. Check the [Go runtime docs](https://vercel.com/docs/functions/runtimes/go) before relying on it for production traffic.

With the CLI:

```bash
npm i -g vercel
vercel login
vercel link
make preview   # preview deployment
make deploy    # production deployment
```

With Git: import the repository in the Vercel dashboard. Leave the Framework Preset on "Other" (the value `framework: null` in `vercel.json`) and keep the root directory at the repository root. Vercel finds `go.mod` and the files in `api/` automatically.

A project that was previously linked in Go server mode keeps that preset until you clear it. Set the Framework Preset to "Other" and redeploy, otherwise Vercel builds `cmd/server` instead of the functions.

Smoke test a deployment:

```bash
BASE_URL=https://your-project.vercel.app ./examples/curl.sh
```

The script checks the greeting and the echo timestamp, not just status codes.

`make dev` runs `vercel dev` to test the real per-file functions locally. Run `vercel link` once first. The plain `go run` loop is faster and is the primary workflow, but both are worth running when you change routing.

## Project layout

```text
.
├── api/                      One function per file, plus its init registration
├── cmd/server/main.go        Local and non-Vercel entrypoint
├── core/                     Registry, response helpers, logging, tests
├── docs/                     OpenAPI spec, Swagger UI page, guides
├── examples/                 HTTP and curl requests
├── vercel.json               framework: null, so Vercel builds the files in api/
├── Makefile
└── go.mod
```

Only `vercel.json` and the file-per-function layout of `api/` are specific to Vercel. The handlers, `core/`, and `docs/` do not know which host runs them. `cmd/server` serves the same handlers from a normal `net/http` server.

## Configuration

| Variable | Used by | Default |
|---|---|---|
| `PORT` | `cmd/server` only | `8080` |
| `LOG_LEVEL` | all | `info` |

Vercel functions do not listen on a port. They receive a request and write a response. `PORT` matters only when you run the binary yourself.

Logs are JSON on stdout. Set env vars in the Vercel dashboard for deployments.

## Use it as a template

1. Change the module path:

   ```bash
   go mod edit -module you/go-serverless
   grep -rl 'haqi/go-serverless' --include='*.go' --include='*.md' . \
     | xargs perl -pi -e 's#haqi/go-serverless#you/go-serverless#g'
   ```

   A module path without a dot works for an application. It is not fetchable
   with `go get`; use a domain-prefixed path if you publish this as a library.

2. Update `info.title` in `docs/openapi.yaml`.
3. Rename the project in Vercel when you link it.
4. Replace the copyright line in `LICENSE`.
5. Run `make test` and `make run`.

## Adding an endpoint

Add a file under `api/` and redeploy. See [docs/extending.md](docs/extending.md) for a worked example and a list of where auth, storage, and config should go later.

## Moving to another host

See [docs/portability.md](docs/portability.md). Docker, Fly.io, Render, Cloud Run, and AWS Lambda recipes are included. The Lambda recipe adds dependencies only to its own adapter.

## Troubleshooting

| Symptom | Cause and fix |
|---|---|
| Vercel builds a server instead of functions | The Framework Preset must be "Other" (`"framework": null`), and `cmd/server` must not be selected as the entrypoint. |
| A new file does not appear on Vercel | Each `api/*.go` file must declare `package handler` and export exactly one function with the `http.HandlerFunc` signature. Redeploy after adding it. |
| Build cannot find the module | `go.mod` must sit at the repository root. |
| `/api/missing` returns HTML instead of the JSON error shape | Vercel answers unknown paths before a function runs. Only `cmd/server` has a JSON catch-all. |
| `vercel dev` returns 502 `NO_RESPONSE_FROM_FUNCTION` | The function failed to build. Read the terminal output. A common cause is an `internal/` import from `api/`, because Vercel rewrites the module to `handler`. |
| `/api/docs` is blank | Swagger UI loads from jsDelivr and needs internet access. Vendor `swagger-ui-dist` into `docs/` if the page must work offline. |
| `/api/docs` renders but Try it out fails | The spec loads from `/api/openapi.yaml`. Check that the deployment serves that path and that the browser can reach the deployment URL. |
| Vercel rejects the Go version | Lower the `go` directive in `go.mod` to a version Vercel supports, such as 1.25. |
| `make portability` fails | Something in `api/`, `core/`, or `docs/` grew a third-party import, or a Go file outside `api/` mentions Vercel. Move host-specific code out of the core. |

## Documentation

- [Architecture](docs/architecture.md)
- [Extending the API](docs/extending.md)
- [Portability](docs/portability.md)
- [OpenAPI spec](docs/openapi.yaml)
- [Agent skills](agents/skills/)

## License

MIT. See [LICENSE](LICENSE).
