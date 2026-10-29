# Extending the API

The walkthrough below adds `GET /api/items?id=abc123`. The endpoint is one file, one registration, and no router edit.

## 1. Add the file

Create `api/items.go`:

```go
package handler

import (
	"net/http"

	core "haqi/go-serverless/core"
	"haqi/go-serverless/lib"
)

func init() {
	core.Register(http.MethodGet, "/api/items", Items)
}

type itemResponse struct {
	ID string `json:"id"`
}

func Items(w http.ResponseWriter, r *http.Request) {
	lib.Handle(w, r, items, http.MethodGet, http.MethodHead)
}

func items(r *http.Request) (any, error) {
	id, err := lib.RequiredQuery(r, "id")
	if err != nil {
		return nil, err
	}
	return itemResponse{ID: id}, nil
}
```

That is the whole registration. Vercel maps `api/items.go` to `/api/items`. `cmd/server` sees the route because the file's `init` ran when the `api` package was imported. There is no central route table to update.

The exported `Items` function is the adapter Vercel and `net/http` call. The unexported `items` function holds the logic and returns the value for the success envelope or an error. It never writes to the response.

Rules for the file:

- Declare `package handler`.
- Export exactly one function with the `http.HandlerFunc` signature. Vercel picks the first one in the file.
- Use a name that no other file in `api/` exports. All files compile as one package.
- Delegate to `lib.Handle` first. It applies the method guard, panic recovery, envelope, and error mapping on every host.
- Keep the logic function unexported and give it the `lib.Handler` signature: `func(*http.Request) (any, error)`.
- Return `lib.BadRequest`, `lib.NotFound`, and the other error constructors. Do not call `http.Error` or write JSON by hand.

## Request, response, and error contract

The wire format is defined once in `lib/`:

```json
{ "data": { "id": "abc123" } }
```

```json
{ "error": { "code": "invalid_request", "message": "id is required" } }
```

- Success bodies always use the `data` key. The docs endpoints are the only raw responses, because they serve HTML and YAML.
- Error bodies always use the `error` key with a stable `code` and a human-readable `message`.
- `lib.Handle` writes 200. Use `lib.HandleStatus(w, r, http.StatusCreated, items, http.MethodPost)` for another 2xx status; a 204 writes no body.
- An error that is not an `*APIError` becomes a 500 with code `internal_error`. Its text is logged, not returned. Return a wrapped plain error for unexpected failures and an `*APIError` when the client should see a specific status.

| Need | Helper | Status | Code |
|---|---|---|---|
| Invalid input | `lib.BadRequest("...")` | 400 | `invalid_request` |
| Missing auth | `lib.Unauthorized("...")` | 401 | `unauthorized` |
| Denied | `lib.Forbidden("...")` | 403 | `forbidden` |
| Missing resource | `lib.NotFound("...")` | 404 | `not_found` |
| State conflict | `lib.Conflict("...")` | 409 | `conflict` |
| Body over the limit | `lib.PayloadTooLarge(maxBytes)` | 413 | `payload_too_large` |
| Semantic validation | `lib.Unprocessable("...")` | 422 | `unprocessable_entity` |
| Rate limit | `lib.TooManyRequests("...")` | 429 | `too_many_requests` |
| Unexpected failure | `lib.Internal(err)` or a plain error | 500 | `internal_error` |

For JSON bodies, use the shared decoder:

```go
func createItem(r *http.Request) (any, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := lib.DecodeJSON(r, &req, 1<<10); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, lib.BadRequest("name is required")
	}
	return itemResponse{ID: req.Name}, nil
}
```

`DecodeJSON` caps the body, rejects unknown fields, rejects trailing values, and returns `*APIError` values ready for `Handle`. A limit of `0` uses `lib.DefaultMaxBodyBytes` (1 MiB), so pass an explicit limit when the endpoint should be tighter.

Use `lib.Query(r, "name", "World")` for an optional parameter and `lib.RequiredQuery(r, "id")` for a required one. Both trim whitespace; the required form returns a 400 when the value is blank.

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
                data:
                  id: abc123
        '400':
          $ref: '#/components/responses/BadRequest'
        '405':
          $ref: '#/components/responses/MethodNotAllowed'
        '500':
          $ref: '#/components/responses/InternalError'
```

Add the schema under `components/schemas`. The schema describes the envelope, not only the payload:

```yaml
    ItemResponse:
      type: object
      required: [data]
      properties:
        data:
          type: object
          required: [id]
          properties:
            id:
              type: string
              example: abc123
```

Every JSON endpoint can return 500 through the adapter, so reference `InternalError` unless the operation cannot fail internally.

## 3. Test

Add a row to the table in `core/api_test.go`. The test package imports `api`, so its `init` registrations run and `core.NewRouter` sees the new route.

```go
{
	name:         "item",
	method:       http.MethodGet,
	path:         "/api/items?id=abc123",
	wantStatus:   http.StatusOK,
	wantContains: `"data":{"id":"abc123"}`,
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

`lib/lib_test.go` covers the shared adapter, decoder, and error mapping. Add a unit test there only when you change `lib/` itself.

## 4. Examples

Add the request to `examples/requests.http`. In `examples/curl.sh`, use the `expect` helper so the check covers the body, not just a 2xx status:

```bash
body=$(curl -fsS "$BASE_URL/api/items?id=abc123")
expect item "$body" '"data":{'
expect item-id "$body" '"id":"abc123"'
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
- Every status code the handler can return is documented, including `405` where the path rejects other methods and `500` for the adapter.
- Success examples are wrapped in `data`; error examples use `code` and `message`.
- Shared responses such as `BadRequest`, `MethodNotAllowed`, and `InternalError` are reused instead of copied.
- Schemas cover required fields and constraints such as `maxLength`.
- Examples use real values the handler accepts.
- Tests cover the happy path and each error response.
- The handler name is unique across `api/`, and the file exports only that one handler.

## Where things go later

This project does not implement these, but each has an obvious home:

| Need | Where | Notes |
|---|---|---|
| Auth | Middleware wrapping the mux in `NewRouter`, plus a check before `lib.Handle` runs the logic | Every host and every function gets it |
| Database | `store/` at the repository root | Keep SQL and drivers out of handlers; pass a store into `NewRouter` when the signature needs it |
| Config | `config/` at the repository root | Read env vars at startup, not inside handlers |
| More entrypoints | New `cmd/<name>` directory | Share code through `core/` and `lib/` |
| Environments | Vercel dashboard env vars | Preview and production have separate values |
| CI | `.github/workflows/ci.yml` | Run `make test` and `make portability`; leave deploys to Vercel's Git integration |
| Observability | Middleware around the mux, or inside `lib.Handle` | `slog` already emits JSON; add spans or request IDs there |

Keep anything that `api/` imports outside `internal/`. Vercel rewrites the module during the function build, and an `internal/` import from `api/` then fails with "use of internal package not allowed". That is why the shared packages are `core/` and `lib/` rather than `internal/api`.
