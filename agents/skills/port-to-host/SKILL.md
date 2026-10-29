---
name: port-to-host
description: Move this API from Vercel to Docker, Fly.io, Render, Cloud Run, Railway, or AWS Lambda. Use when deploying to another host or adding a host adapter.
---

# Port to another host

The handlers in `api/`, the HTTP contract in `lib/`, and the registry in `core/` know nothing about Vercel. Only `vercel.json` and the file-per-function layout are host-specific. `cmd/server` reads `PORT`, binds `:PORT`, blank-imports `api/`, and serves every registered route under `/api`.

Other hosts use the server, not the per-file functions. The `api/` files still compile and register; Vercel simply happens to call them one by one.

## Checklist

1. Delete or ignore `vercel.json`.
2. Build `./cmd/server` for the target OS and architecture.
3. Set `PORT` if the platform requires a specific value.
4. Set `LOG_LEVEL` if you need more or less log volume.
5. Point health checks at `/api/health`.
6. Run the smoke test against the deployed URL.
7. Update the README deploy section so it names the real host.

## Docker

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

If a `go.sum` appears, copy it with `go.mod` before `go mod download`.

```bash
docker build -t go-serverless .
docker run --rm -p 8080:8080 go-serverless
curl localhost:8080/api/health
```

The image is static and runs as a non-root user.

## Platform notes

```bash
make build
PORT=8080 ./bin/server
```

- Fly.io: deploy the Docker image or run `fly launch` with the Dockerfile.
- Render: build with `make build`, start with `./bin/server`.
- Cloud Run: deploy the container image and let it set `PORT`.
- Railway: deploy the repository with the Dockerfile.
- Any VM: copy `bin/server`, set `PORT`, run it under a process manager.

## AWS Lambda

Lambda needs an adapter and two dependencies. Keep both out of `core/` and `lib/`. Add `cmd/lambda/main.go`:

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

Deploy with AWS SAM using `provided.al2023`, `arm64`, and an HTTP API with payload v2. Add `ANY /{proxy+}` and `ANY /` routes. Check the current `httpadapter` constructor name before copying the snippet.

The `make portability` guard still passes because it checks dependencies under `api/`, `core/`, `lib/`, and `docs/`, and forbids Vercel references in Go files outside `api/`.

## Verify

```bash
BASE_URL=https://your-host.example ./examples/curl.sh
```

The script checks `/api/health`, `/api/hello?name=Go`, `/api/echo`, the 400 error shape, `/api/docs`, and `/api/openapi.yaml`. A host that passes it serves the same API as Vercel, including the `data` success envelope.

## Rules

- Put host-specific code in a new `cmd/<name>` directory, never in `core/` or `lib/`.
- Do not add a third-party dependency to `api/`, `core/`, `lib/`, or `docs/`.
- Anything imported by `api/` must stay outside `internal/` so Vercel's module rewrite can still import it.
- Do not change route paths during a move. Clients and the OpenAPI spec expect the `/api` prefix.
- If the new host cannot serve the CDN, vendor `swagger-ui-dist` into `docs/` and update `docs/index.html`.
