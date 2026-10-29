---
name: add-endpoint
description: Add or change an HTTP endpoint in this Go API. Use when adding routes, handlers, validation, response types, tests, OpenAPI paths, or example requests under /api.
---

# Add an endpoint

Every route is served under `/api` by `cmd/server`. A complete change touches the handler, router, tests, spec, and examples.

## 1. Handler

Add the handler to `internal/api/handlers.go`. Reuse `writeJSON` and `writeError` so the JSON and error shapes stay consistent.

```go
type itemResponse struct {
	ID string `json:"id"`
}

func item(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, itemResponse{ID: r.PathValue("id")})
}
```

For JSON bodies, copy the `echo` pattern: cap the body with `http.MaxBytesReader`, decode with `DisallowUnknownFields`, reject trailing content, then validate fields.

## 2. Route

Add one entry to `routes` in `internal/api/router.go`:

```go
{http.MethodGet, "/api/items/{id}", item},
```

`NewRouter` derives the 405 handler and the `HEAD` allowance from this table. Do not register the route anywhere else.

## 3. Test

Add rows to the table in `internal/api/api_test.go`. Cover success, each error status, and 405 when the route rejects other methods.

```go
{
	name:         "item",
	method:       http.MethodGet,
	path:         "/api/items/abc123",
	wantStatus:   http.StatusOK,
	wantContains: `"id":"abc123"`,
},
```

## 4. Spec

Add the path to `docs/openapi.yaml` with every status the handler can return. Reuse `#/components/responses/MethodNotAllowed` instead of writing another 405 block.

```yaml
  /api/items/{id}:
    get:
      tags: [examples]
      operationId: getItem
      summary: Return an item id
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
          example: abc123
      responses:
        '200':
          description: The requested item.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ItemResponse'
              example:
                id: abc123
        '405':
          $ref: '#/components/responses/MethodNotAllowed'
```

Add the schema under `components/schemas`:

```yaml
    ItemResponse:
      type: object
      required: [id]
      properties:
        id:
          type: string
          example: abc123
```

## 5. Examples

Add the request to `examples/requests.http`. In `examples/curl.sh`, use the `expect` helper so the check covers the body:

```bash
body=$(curl -fsS "$BASE_URL/api/items/abc123")
expect item "$body" '"id":"abc123"'
```

## 6. Verify

```bash
gofmt -l .
make test
make portability
npx --yes @redocly/cli@latest lint docs/openapi.yaml
```

Start the server and run the smoke test:

```bash
PORT=18080 go run ./cmd/server
BASE_URL=http://localhost:18080 ./examples/curl.sh
```

Open `http://localhost:18080/api/docs` and confirm the operation appears and Try it out returns the expected response.

## Rules

- Keep core code stdlib-only. No new dependency belongs in `internal/` or `docs/`.
- Do not add a route outside `/api`.
- Keep the spec and the handler in sync. `docs/extending.md` has the full drift checklist.
