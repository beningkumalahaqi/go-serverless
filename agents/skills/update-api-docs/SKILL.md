---
name: update-api-docs
description: Update README and docs for this Go API after changing routes, paths, status codes, deployment, or behavior. Use when docs or docs/openapi.yaml drift from the code.
---

# Update the docs

The code and `Makefile` are the source of truth. Docs must describe what the repository does today, not what an earlier deployment did.

## Find the stale parts

```bash
grep -rn '/api/' README.md docs examples internal cmd
grep -rn 'health\|hello\|echo\|docs\|openapi' README.md docs examples
```

Check every path against the route table in `internal/api/router.go`.

## Checklist

1. **README endpoints table** lists the real methods and paths, all under `/api`.
2. **README status codes** match the handlers: 400, 404, 405 with `Allow`, 413.
3. **README local commands** use `/api/health`, `/api/hello`, `/api/echo`, and `/api/docs`.
4. **README deploy section** matches `vercel.json`. Current mode is `"framework": "go"` with no `api/` function directory and no rewrites.
5. **`docs/openapi.yaml` paths** include the `/api` prefix and document every status the handler can return.
6. **`docs/index.html`** points Swagger UI at `/api/openapi.yaml`.
7. **`docs/architecture.md`** matches the current entrypoint, route prefix, and failure behavior.
8. **`docs/extending.md`** examples use `/api/...` paths.
9. **`docs/portability.md`** lists only `vercel.json` as host-specific and uses `/api/health` in the host checklist.
10. **`examples/curl.sh` and `examples/requests.http`** use the same paths as the router.

## OpenAPI rules

- Reuse `#/components/responses/MethodNotAllowed` for 405 responses.
- Reuse `#/components/responses/BadRequest` and `PayloadTooLarge` for `/api/echo`.
- Keep `servers: [{ url: / }]`, so a relative deployment serves the spec at any origin.
- Keep `security: []` while the API has no authentication.

Lint after editing:

```bash
npx --yes @redocly/cli@latest lint docs/openapi.yaml
```

The spec must be valid with zero errors and zero warnings.

## Verify the rendered docs

```bash
PORT=18080 go run ./cmd/server
```

Open `http://localhost:18080/api/docs`. Confirm every operation appears, the Try it out request URL includes `/api`, and no response is missing from the spec.

Do not claim a deployment is verified unless `examples/curl.sh` was run against the deployed URL. Local `go run` cannot prove Vercel routing.

## Related files

- `README.md`
- `docs/architecture.md`
- `docs/extending.md`
- `docs/portability.md`
- `docs/openapi.yaml`
- `docs/index.html`
