---
name: update-api-docs
description: Update README and docs for this Go API after changing routes, paths, status codes, response shapes, deployment, or behavior. Use when docs or docs/openapi.yaml drift from the code.
---

# Update the docs

The code and `Makefile` are the source of truth. Docs must describe what the repository does today, not what an earlier deployment did.

## Find the stale parts

```bash
grep -rn 'core.Register' api
grep -rn 'lib.Handle\|lib.BadRequest\|lib.DecodeJSON\|lib.HandleStatus' api
grep -rn '/api/' README.md docs examples cmd
grep -rn 'health\|hello\|echo\|docs\|openapi' README.md docs examples
```

Every route is registered by an `init` function in `api/`, and the URL is the file name. Check each documented path against the files:

```bash
for f in api/*.go; do printf '%s -> /api/%s\n' "$f" "$(basename "$f" .go)"; done
```

Remember that `api/openapi.yaml.go` maps to `/api/openapi.yaml`.

## Checklist

1. **README endpoints table** lists the real methods and paths, all under `/api`.
2. **README response shapes** show both envelopes: `{"data": ...}` for 2xx JSON and `{"error": {"code", "message"}}` for failures. Mention that `/api/docs` and `/api/openapi.yaml` are raw responses.
3. **README status codes** match the handlers: 400, 404, 405 with `Allow`, 413, and 500 for unexpected failures. Note that Vercel returns its own HTML 404 for unknown paths and that 500 causes are logged, not returned.
4. **README local commands** use `/api/health`, `/api/hello`, `/api/echo`, and `/api/docs`.
5. **README deploy section** matches `vercel.json`. Current mode is `"framework": null` with one function per file in `api/`, no rewrites, and no `cmd/server` in production.
6. **README project layout** lists `lib/` and keeps `core/` described as registry, router, logging, and tests.
7. **`docs/openapi.yaml` paths** include the `/api` prefix and document every status the handler can return, including 500 for JSON endpoints.
8. **`docs/openapi.yaml` schemas** describe the success envelope (`data`) and reuse `Error` for failures.
9. **`docs/index.html`** points Swagger UI at `/api/openapi.yaml`.
10. **`docs/architecture.md`** matches the current entrypoints, registry, `lib.Handle` adapter, envelope, package name, and failure behavior.
11. **`docs/extending.md`** describes adding one file under `api/`, including `lib.Handle`, `package handler`, and unique-name rules.
12. **`docs/portability.md`** names `vercel.json` and the `api/` layout as host-specific, lists `lib/`, and uses `/api/health` in the host checklist.
13. **`examples/curl.sh` and `examples/requests.http`** use the same paths as the files in `api/`, and the curl checks include the `data` envelope and the error code.
14. **Module path** in README and docs snippets matches `go.mod`, including the template commands that rename it.

## Contract checks

Search for drift between the code and the documented wire format:

```bash
grep -rn 'lib.HandleStatus\|lib.Respond\|lib.Created\|lib.NoContent' api
grep -rn '"data"' README.md docs examples
grep -rn 'internal_error' README.md docs lib
```

If a handler writes JSON itself instead of returning a value through `lib.Handle`, either change the handler or document why it is a raw endpoint. Only the docs endpoints are raw today.

## OpenAPI rules

- Reuse `#/components/responses/MethodNotAllowed` for 405 responses.
- Reuse `#/components/responses/BadRequest`, `PayloadTooLarge`, and `InternalError` where they apply.
- Every success schema wraps its payload in `data`; every success example shows the wrapper.
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

Open `http://localhost:18080/api/docs`. Confirm every operation appears, the Try it out request URL includes `/api`, every success example shows `data`, and no response is missing from the spec.

When routing changed, also render the page through the real function builder:

```bash
vercel dev --listen 127.0.0.1:18089
```

Open `http://127.0.0.1:18089/api/docs`. A local `go run` cannot prove Vercel routing.

Do not claim a deployment is verified unless `examples/curl.sh` was run against the deployed URL.

## Related files

- `README.md`
- `docs/architecture.md`
- `docs/extending.md`
- `docs/portability.md`
- `docs/openapi.yaml`
- `docs/index.html`
- `api/`
- `lib/`
