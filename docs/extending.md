# Extending the API

The walkthrough below adds `GET /items/{id}`. It touches every file a new endpoint needs.

## 1. Handler

Add to `internal/api/handlers.go`:

```go
type itemResponse struct {
	ID string `json:"id"`
}

func item(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, itemResponse{ID: r.PathValue("id")})
}
```

`r.PathValue` works because Go 1.22+ `ServeMux` patterns can capture path segments.

## 2. Route

Add one entry to the `routes` slice in `internal/api/router.go`:

```go
{http.MethodGet, "/items/{id}", item},
```

The router registers the method pattern and a bare pattern for 405 responses automatically. `GET` routes also accept `HEAD`.

## 3. Spec

Add the path to `docs/openapi.yaml`:

```yaml
  /items/{id}:
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

## 4. Test

Add a row to the table in `internal/api/api_test.go`:

```go
{
	name:         "item",
	method:       http.MethodGet,
	path:         "/items/abc123",
	wantStatus:   http.StatusOK,
	wantContains: `"id":"abc123"`,
},
```

Add a failure case too, such as a wrong method returning 405, if the route introduces one.

## 5. Examples

Add the request to `examples/requests.http` and `examples/curl.sh`:

```bash
curl -fsS "$BASE_URL/items/abc123"
```

## 6. Run

```bash
make test
make run
```

Open `/docs` and confirm the new operation appears and "Try it out" returns the expected response.

## Spec drift checklist

Run through this before opening a pull request:

- Every route in `router.go` has a path in `openapi.yaml`.
- Method, path parameters, query parameters, and request bodies match the handler.
- Every status code the handler can return is documented.
- Schemas cover required fields and constraints such as `maxLength`.
- Examples use real values the handler accepts.
- Tests cover the happy path and each error response.

## Where things go later

This project does not implement these, but each has an obvious home:

| Need | Where | Notes |
|---|---|---|
| Auth | Middleware in `internal/api` wrapping the mux in `NewRouter` | Both adapters get it automatically |
| Database | `internal/store` | Keep SQL and drivers out of handlers; pass a store into `NewRouter` when the signature needs it |
| Config | `internal/config` | Read env vars at startup, not inside handlers |
| More functions | New file in `api/` and/or `cmd/<name>` | Share code through `internal/` |
| Environments | Vercel dashboard env vars | Preview and production have separate values |
| CI | `.github/workflows/ci.yml` | Run `make test` and `make portability`; leave deploys to Vercel's Git integration |
| Observability | Middleware around the mux | `slog` already emits JSON; add spans or request IDs there |

Keep the core portable: anything host-specific belongs in an adapter, never in `internal/`.
