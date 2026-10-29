---
name: deploy-vercel
description: Deploy this API to Vercel as per-file Go functions and verify it. Use for preview or production releases, redeploys, and deployed path or 404 problems.
---

# Deploy to Vercel

Vercel builds every `.go` file in `api/` as its own function. `vercel.json` disables the Go framework preset:

```json
{
  "$schema": "https://openapi.vercel.sh/vercel.json",
  "framework": null
}
```

With no framework selected, Vercel's zero-config builder maps `api/health.go` to `/api/health` and `api/openapi.yaml.go` to `/api/openapi.yaml`. `cmd/server` is not used.

Each file declares `package handler` and exports exactly one `http.HandlerFunc`. Vercel rejects `package main` when `go.mod` is present, and it picks the first exported handler in each file. The `init` registration is for `cmd/server`; Vercel calls the handler directly.

Vercel's Go runtime is in beta. Mention that when the task is production traffic.

## Preflight

Run the local gate first:

```bash
gofmt -l .
make test
make portability
```

Do not deploy a failing tree.

When routing, a filename, or the module path changed, build the function bundles locally without deploying:

```bash
vercel build --yes
find .vercel/output/functions -name '*.func' | sort
```

There must be one `.func` per file in `api/`. This needs a linked project, so run `vercel link` once first. Add `vercel dev` when you also want to exercise the routes.

## Deploy

```bash
npm i -g vercel
vercel login
vercel link
make preview   # vercel deploy
make deploy    # vercel deploy --prod
```

With Git, push and let the Vercel integration build. No build settings are needed. The Framework Preset must be "Other" (`framework: null`).

A project previously linked in Go server mode keeps that preset until you clear it. Set the Framework Preset to "Other" and redeploy, otherwise Vercel builds `cmd/server` instead of the functions.

## Verify

```bash
export BASE_URL=https://your-project.vercel.app
./examples/curl.sh
curl -sS -D - -o /dev/null -X POST "$BASE_URL/api/health" | grep -i '^allow'
curl -sS -o /dev/null -w 'missing: %{http_code}\n' "$BASE_URL/api/missing"
```

Expected results:

| Request | Result |
|---|---|
| `GET /api/health` | `{"status":"ok"}` |
| `GET /api/hello?name=Go` | `{"message":"Hello, Go!"}` |
| `POST /api/echo` | message plus a UTC `receivedAt` |
| `POST /api/echo` with `{}` | 400, `invalid_request` |
| `POST /api/health` | 405 with `Allow: GET, HEAD` |
| `GET /api/echo` | 405 with `Allow: POST` |
| `GET /api/openapi.yaml` | the spec |
| `GET /api/docs` | Swagger UI |
| `GET /` and `GET /api` | the platform's HTML 404 |

Do not report the deployment as working until the script passes and the 405 checks return the JSON error shape.

## Troubleshooting

| Symptom | Cause and fix |
|---|---|
| Every path returns 404 | No function matched. Confirm each file is directly in `api/`, declares `package handler`, and exports a handler. Redeploy. |
| A new endpoint is missing | The file was added after the last deploy. Redeploy; discovery happens at build time. |
| Vercel builds a server instead of functions | The Framework Preset is still `go`. Set it to "Other" so `"framework": null` applies. |
| A function returns 502 `NO_RESPONSE_FROM_FUNCTION` | The function failed to build or panic. Check the build log. Importing `internal/...` from `api/` fails because Vercel rewrites the module to `handler`. |
| `/api` returns 404 | Expected. There is no `api.go`; the endpoints are files beneath `api/`. |
| `/api/missing` returns HTML | Expected. Unknown paths never reach a function, so the platform 404 is returned. `cmd/server` is the only place with the JSON catch-all. |
| Wrong method returns 405 | Correct. Handlers call `core.Allow`. |
| `/api/docs` is blank | Swagger UI loads from jsDelivr and needs internet access. |
| Try it out fails | The page points at `/api/openapi.yaml`; confirm that file deployed and is reachable. |
| Build cannot find the module | `go.mod` must sit at the repository root. |
| Go version rejected | Adjust the `go` directive in `go.mod` to a version Vercel accepts. |

## Constraints

- Do not add `rewrites` or `routes` to `vercel.json`. The file name is the route.
- Do not set `"framework": "go"`. That selects server mode and ignores `api/`.
- Keep the directory flat. Files in subdirectories of `api/` map to different URLs.
- Keep Vercel-specific changes limited to `vercel.json` and the deploy section of the README.
