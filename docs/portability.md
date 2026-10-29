# Portability

Only `vercel.json` and the file-per-function layout of `api/` know about Vercel. The rest is a normal Go HTTP server:

| File | Role |
|---|---|
| `api/` | One handler per file; compiled into `cmd/server` too |
| `core/` | Registry, router, logging, tests |
| `lib/` | Request decoder, success envelope, error type, handler adapter |
| `cmd/server` | Reads `PORT`, serves the router |
| `docs/` | Embedded spec and Swagger UI |
| `examples/`, `Makefile` | Local workflow |

Delete `vercel.json` and the rest runs anywhere Go runs. `cmd/server` blank-imports `api/`, so every handler still registers and the same URLs are served. Vercel functions never bind a port; every host below uses the server instead. The recipes are starting points; the Docker and Lambda snippets are not exercised by this repository's tests.

## Docker

Multi-stage build with a static binary and a distroless runtime:

```dockerfile
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]
```

```bash
docker build -t go-serverless .
docker run --rm -p 8080:8080 go-serverless
curl localhost:8080/api/health
```

`cmd/server` reads `PORT`, so every platform below can inject its own value.

## Fly.io, Render, Cloud Run, Railway

Build the binary and run it:

```bash
make build
PORT=8080 ./bin/server
```

- Fly.io: deploy the Docker image or use `fly launch` with the Dockerfile above.
- Render: create a web service from the repository, set the build command to `make build` and the start command to `./bin/server`.
- Cloud Run: deploy the container image and let it set `PORT`.
- Railway: deploy the repository with the Dockerfile.

## AWS Lambda

Lambda needs one adapter file and two dependencies that stay out of the shared packages. Add `cmd/lambda/main.go`:

```go
package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"

	"haqi/go-serverless/core"
)

func main() {
	lambda.Start(httpadapter.NewV2(core.NewRouter()).ProxyWithContext)
}
```

```bash
go get github.com/aws/aws-lambda-go github.com/awslabs/aws-lambda-go-api-proxy
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap ./cmd/lambda
```

Deploy with AWS SAM using the `provided.al2023` runtime and an HTTP API (payload v2). The function handler is the `bootstrap` binary. A minimal SAM template needs:

- `Runtime: provided.al2023`
- `Architectures: [arm64]`
- `Handler: bootstrap`
- `BuildMethod: makefile` with a `build-<FunctionLogicalId>` target that writes the binary to `$(ARTIFACTS_DIR)/bootstrap`
- `ANY /{proxy+}` and `ANY /` routes

Check the current `httpadapter` constructor name before copying the snippet; the library has changed it across releases. If you would rather avoid the dependency, a payload v2 adapter is roughly 40 lines and needs no third-party packages.

The `make portability` guard checks two things: `api/`, `core/`, `lib/`, and `docs/` must not import third-party modules, and no Go file outside `api/` may mention Vercel. The Lambda adapter lives in `cmd/lambda`, so its dependencies never reach the shared packages.

## Host checklist

When moving:

1. Delete or ignore `vercel.json`.
2. Build `./cmd/server` for the target OS and architecture.
3. Set `PORT` if the platform requires a specific value.
4. Set `LOG_LEVEL` if you need more or less log volume.
5. Point health checks at `/api/health`.
6. Run `examples/curl.sh` against the deployed URL with `BASE_URL` set.
