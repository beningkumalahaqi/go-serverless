---
name: deploy-vercel
description: Deploy this API to Vercel in Go server mode and verify it. Use for preview or production releases, redeploys, and deployed path or 404 problems.
---

# Deploy to Vercel

Vercel runs `cmd/server` as a Go server. `vercel.json` contains only the framework preset:

```json
{
  "$schema": "https://openapi.vercel.sh/vercel.json",
  "framework": "go"
}
```

The Go preset detects `go.mod` at the root and one of `main.go`, `cmd/api/main.go`, or `cmd/server/main.go`. This repository uses `cmd/server/main.go`. The server must listen on `PORT`, which it does.

There is no `api/` directory and no rewrite. All routes already live under `/api`, so `/api/health` is a normal server path.

Vercel's Go runtime is in beta. Mention that when the task is production traffic.

## Preflight

Run the local gate first:

```bash
gofmt -l .
make test
make portability
```

Do not deploy a failing tree.

## Deploy

```bash
npm i -g vercel
vercel login
vercel link
make preview   # vercel deploy
make deploy    # vercel deploy --prod
```

With Git, push and let the Vercel integration build. No build settings are needed beyond the framework preset.

If the project was linked while it still had an `api/` function, clear any Framework Preset override in the project settings and redeploy. A stale deployment keeps the old routes.

## Verify

```bash
BASE_URL=https://your-project.vercel.app ./examples/curl.sh
```

Expected results:

| Request | Result |
|---|---|
| `GET /api/health` | `{"status":"ok"}` |
| `GET /api/hello?name=Go` | `{"message":"Hello, Go!"}` |
| `POST /api/echo` | message plus a UTC `receivedAt` |
| `POST /api/echo` with `{}` | 400, `invalid_request` |
| `GET /api/openapi.yaml` | the spec |
| `GET /api/docs` | Swagger UI |
| `GET /` and `GET /api` | 404 by design |

Do not report the deployment as working until the script passes.

## Troubleshooting

| Symptom | Cause and fix |
|---|---|
| Every path returns 404 | The deployment is stale or the framework preset is not `go`. Confirm `framework: "go"` and redeploy. |
| `/api` returns 404 | Expected. There is no route at `/api`; the endpoints are beneath it. |
| Old paths such as `/health` return 404 | Expected. Clients must use `/api/health` and the other prefixed paths. |
| Vercel builds a function instead of the server | An `api/` directory reappeared, or the framework preset is overridden. Remove the directory and set the preset to `go`. |
| `/api/docs` is blank | Swagger UI loads from jsDelivr and needs internet access. |
| Try it out fails | The page points at `/api/openapi.yaml`; confirm that path is deployed and reachable. |
| Build cannot find the module | `go.mod` must sit at the repository root. |
| Go version rejected | Adjust the `go` directive in `go.mod` to a version Vercel accepts. |

## Constraints

- Do not add `rewrites` or `routes` to `vercel.json` for path handling. Server mode serves the paths directly.
- Do not recreate `api/index.go`. That mode caused the original `/api` path collision.
- Keep Vercel-specific changes limited to `vercel.json` and the deploy section of the README.
