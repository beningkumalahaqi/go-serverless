# Extending the API

The walkthrough below adds `GET /api/items?id=abc123`. The endpoint is one file, one registration, and no router edit.

## 1. Add the file

Create `api/items.go`:

```go
package handler

import (
	"net/http"

	core "haqi/go-serverless/core"
)

func init() {
	core.Register(http.MethodGet, "/api/items", Items)
}

type itemResponse struct {
	ID string `json:"id"`
}

func Items(w http.ResponseWriter, r *http.Request) {
	if !core.Allow(w, r, http.MethodGet, http.MethodHead) {
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		core.WriteError(w, http.StatusBadRequest, "invalid_request", "id is required")
		return
	}
	core.WriteJSON(w, http.StatusOK, itemResponse{ID: id})
}
```

That is the whole registration. Vercel maps `api/items.go` to `/api/items`. `cmd/server` sees the route because the file's `init` ran when the `api` package was imported. There is no central route table to update.

Rules for the file:

- Declare `package handler`.
- Export exactly one function with the `http.HandlerFunc` signature. Vercel picks the first one in the file.
- Use a name that no other file in `api/` exports. All files compile as one package.
- Call `core.Allow` first. A deployed function receives every method sent to its path.
- Reuse `core.WriteJSON` and `core.WriteError` so the response shapes stay consistent.

For JSON bodies, copy the `Echo` handler in `api/echo.go`: cap the body with `http.MaxBytesReader`, decode with `DisallowUnknownFields`, reject trailing content, then validate fields.

Dynamic URL segments such as `/api/items/{id}` do not arrive as `r.PathValue`. Vercel routes the request to the file and passes the raw path, so read the id from the query string or parse `r.URL.Path` yourself.

## 2. Spec

Add the path to `docs/openapi.yaml`:

```yaml
  /api/items:
    get:
      tags: [examples]
      operationId: getItem
      summary: Return an item id
      parameters:
        - name: id
          in: query
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
        '400':
          $ref: '#/components/responses/BadRequest'
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

## 3. Test

Add a row to the table in `core/api_test.go`. The test package imports `api`, so its `init` registrations run and `core.NewRouter` sees the new route.

```go
{
	name:         "item",
	method:       http.MethodGet,
	path:         "/api/items?id=abc123",
	wantStatus:   http.StatusOK,
	wantContains: `"id":"abc123"`,
},
{
	name:          "item requires an id",
	method:        http.MethodGet,
	path:          "/api/items",
	wantStatus:    http.StatusBadRequest,
	wantErrorCode: "invalid_request",
},
```

Add the handler to `TestHandlersGuardMethods` so the wrong-method guard is covered without the local router:

```go
{"items", handler.Items, http.MethodPost, "GET, HEAD"},
```

## 4. Examples

Add the request to `examples/requests.http`. In `examples/curl.sh`, use the `expect` helper so the check covers the body, not just a 2xx status:

```bash
body=$(curl -fsS "$BASE_URL/api/items?id=abc123")
expect item "$body" '"id":"abc123"'
```

## 5. Run

```bash
make test
make run
vercel dev
```

Open `/api/docs` and confirm the new operation appears and "Try it out" returns the expected response. Then call the same path through `vercel dev` to confirm the file mapping.

## Spec drift checklist

Run through this before opening a pull request:

- Every file in `api/` has a path in `openapi.yaml`, and the path matches the file name.
- Method, query parameters, and request bodies match the handler.
- Every status code the handler can return is documented, including `405` where the path rejects other methods.
- Shared responses such as `MethodNotAllowed` are reused instead of copied.
- Schemas cover required fields and constraints such as `maxLength`.
- Examples use real values the handler accepts.
- Tests cover the happy path and each error response.
- The handler name is unique across `api/`, and the file exports only that one handler.

## Where things go later

This project does not implement these, but each has an obvious home:

| Need | Where | Notes |
|---|---|---|
| Auth | Middleware in `core/` wrapping the mux in `NewRouter`, plus a guard called by each function | Every host and every function gets it |
| Database | `store/` at the repository root | Keep SQL and drivers out of handlers; pass a store into `NewRouter` when the signature needs it |
| Config | `config/` at the repository root | Read env vars at startup, not inside handlers |
| More entrypoints | New `cmd/<name>` directory | Share code through `core/` |
| Environments | Vercel dashboard env vars | Preview and production have separate values |
| CI | `.github/workflows/ci.yml` | Run `make test` and `make portability`; leave deploys to Vercel's Git integration |
| Observability | Middleware around the mux | `slog` already emits JSON; add spans or request IDs there |

Keep anything that `api/` imports outside `internal/`. Vercel rewrites the module during the function build, and an `internal/` import from `api/` then fails with "use of internal package not allowed". That is why the shared package is `core/` rather than `internal/api`.
