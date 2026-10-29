# Go Serverless API

A small Go HTTP API that deploys to Vercel as a serverless function and runs unchanged on any host that can execute a binary.

- Three example endpoints, a shared JSON error shape, and tests.
- An OpenAPI 3.0 spec and Swagger UI served by the app itself.
- One router shared by the Vercel adapter and the standalone server.

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

Open <http://localhost:8080/docs> for Swagger UI. The same calls without a browser:

```bash
curl localhost:8080/health
curl 'localhost:8080/hello?name=Go'
curl -X POST localhost:8080/echo \
  -H 'Content-Type: application/json' \
  -d '{"message":"ping"}'
```

`make run` is plain `go run ./cmd/server`.

Stop the server with Ctrl+C.

## Test

```bash
make test        # go vet + go test -race
make portability # core packages stay stdlib-only and free of Vercel imports
```

Both commands must pass before a change is done.

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Liveness check |
| GET | `/hello?name=World` | Greeting, `name` optional |
| POST | `/echo` | Echo a JSON `message` (max 1 KB) with a UTC timestamp |
| GET | `/docs` | Swagger UI |
| GET | `/openapi.yaml` | OpenAPI 3.0.3 document |

Errors use one shape:

```json
{ "error": { "code": "invalid_request", "message": "message is required" } }
```

Status codes: `200`, `400` for bad input, `404` for unknown routes, `405` with an `Allow` header for wrong methods, `413` for a body over 1 KB.

## Deploy to Vercel

The default setup builds `api/index.go` as one Vercel Function. A route sends every path to that function and restores the path it observes, so `/health` reaches the router as `/health`.

Vercel's Go runtime is in beta. Check the [Go runtime docs](https://vercel.com/docs/functions/runtimes/go) before relying on it for production traffic.

With the CLI:

```bash
npm i -g vercel
vercel login
vercel link
make preview   # preview deployment
make deploy    # production deployment
```

With Git: import the repository in the Vercel dashboard. Vercel finds `go.mod` at the root and the function in `api/`. No build settings are needed.

Smoke test a deployment:

```bash
BASE_URL=https://your-project.vercel.app ./examples/curl.sh
```

Local `go run` has no Vercel routing in front of it, so deployed paths can fail while local requests work. Run this script against the deployment after any `vercel.json` change; it checks the greeting and the echo timestamp, not just the status code.

`make dev` runs `vercel dev` if you want to test against the local Vercel emulator. The plain `go run` loop is faster and is the primary workflow.

### Switching to Vercel's Go server mode

The `/api` function is the default because it matches the usual serverless shape. If you would rather deploy `cmd/server` as a long-running Go server, Vercel supports that too:

1. Delete `api/`.
2. Replace `vercel.json` with `{ "framework": "go" }`.

Vercel detects `cmd/server/main.go` and runs it on `PORT`. Everything in `internal/` is untouched. Revert by restoring both files.

## Project layout

```text
.
├── api/index.go              Vercel Function adapter
├── cmd/server/main.go        Standalone server for local use and any host
├── internal/api/             Router, handlers, logging, tests
├── docs/                     OpenAPI spec, Swagger UI page, guides
├── examples/                 HTTP and curl requests
├── vercel.json               Rewrite to the function
├── Makefile
└── go.mod
```

Only `api/` and `vercel.json` are specific to Vercel. The router, handlers, and docs do not know which host runs them.

## Configuration

| Variable | Used by | Default |
|---|---|---|
| `PORT` | `cmd/server` | `8080` |
| `LOG_LEVEL` | all | `info` |

Logs are JSON on stdout. Set env vars in the Vercel dashboard for deployments.

## Use it as a template

1. Change the module path:

   ```bash
   go mod edit -module github.com/you/go-serverless
   grep -rl 'github.com/haqi/go-serverless' --include='*.go' . \
     | xargs perl -pi -e 's#github.com/haqi/go-serverless#github.com/you/go-serverless#g'
   ```

2. Update `info.title` in `docs/openapi.yaml`.
3. Rename the project in Vercel when you link it.
4. Run `make test` and `make run`.

## Adding an endpoint

See [docs/extending.md](docs/extending.md) for a worked example and a list of where auth, storage, and config should go later.

## Moving to another host

See [docs/portability.md](docs/portability.md). Docker, Fly.io, Render, Cloud Run, and AWS Lambda recipes are included. The Lambda recipe adds dependencies only to its own adapter.

## Troubleshooting

| Symptom | Cause and fix |
|---|---|
| Every path returns 404 after deploy | The function is seeing `/api` instead of the original path. Check the `routes` transform in `vercel.json`; if transforms are unavailable, switch to Go server mode. |
| Build cannot find the module | `go.mod` must sit at the repository root. |
| `/docs` is blank | Swagger UI loads from jsDelivr and needs internet access. Vendor `swagger-ui-dist` into `docs/` if the page must work offline. |
| Vercel rejects the Go version | Lower the `go` directive in `go.mod` to a version Vercel supports, such as 1.25. |
| `make portability` fails | Something in `internal/` or `docs/` grew a third-party import or a Vercel reference. Move host code into the adapter. |

## Documentation

- [Architecture](docs/architecture.md)
- [Extending the API](docs/extending.md)
- [Portability](docs/portability.md)
- [OpenAPI spec](docs/openapi.yaml)
