---
name: add-endpoint
description: Add or change an HTTP endpoint in this Go API. Use when adding a file under api/, a lib.Handle adapter, validation, response types, tests, OpenAPI paths, or example requests.
---

# Add an endpoint

Every route is one file in `api/`. The file name becomes the URL, and an `init` function registers the same route with the local server. No central route table exists.

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

That is the whole route change. Vercel maps `api/items.go` to `/api/items`; `cmd/server` sees it because the package was imported.

Rules:

- Declare `package handler`.
- Export exactly one function with the `http.HandlerFunc` signature. Vercel picks the first exported one in the file.
- Use a handler name that is unique across `api/`. All files compile as one package locally.
- Make the exported handler a thin adapter: `lib.Handle(w, r, logic, methods...)`.
- Keep the logic function unexported with the signature `func(*http.Request) (any, error)`. It must not take or use `http.ResponseWriter`.
- Return the success payload as `any`; `lib.Handle` writes `{"data": ...}` with a 200.
- Use `lib.HandleStatus(w, r, http.StatusCreated, logic, methods...)` for another 2xx status. A 204 writes no body.
- Return `lib.BadRequest`, `lib.NotFound`, and the other `lib` constructors for expected failures. Never call `http.Error` or write error JSON by hand.
- Return a plain or wrapped error for unexpected failures; the adapter turns it into a logged 500.
- Keep anything that `api/` imports outside `internal/`; Vercel rewrites the module and rejects internal imports.

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

`lib.DecodeJSON` caps the body, rejects unknown fields, rejects trailing values, and returns `*APIError` values ready for the adapter. Pass a positive limit; `0` means `lib.DefaultMaxBodyBytes` (1 MiB).

Use `lib.Query(r, "name", "World")` for an optional query parameter and `lib.RequiredQuery(r, "id")` for a required one.

Dynamic segments such as `/api/items/{id}` do not populate `r.PathValue` on Vercel. Read the id from the query string or parse `r.URL.Path`.

## 2. Test

Add rows to the table in `core/api_test.go`. The test package imports `api` for its side effects, so the new registration reaches `core.NewRouter`.

```go
{
	name:         "item",
	method:       http.MethodGet,
	path:         "/api/items?id=abc123",
	wantStatus:   http.StatusOK,
	wantContains: `"data":{"id":"abc123"}`,
},
```

Add the handler to `TestHandlersGuardMethods` so the wrong-method guard is covered without the local router:

```go
{"items", handler.Items, http.MethodPost, "GET, HEAD"},
```

The success assertion includes the `data` envelope, so a regression in the shared response contract fails the route test.

`lib/lib_test.go` covers the shared adapter, decoder, and error mapping. Touch it only when changing `lib/` itself.

## 3. Spec

Add the path to `docs/openapi.yaml` with every status the handler can return. Reuse `#/components/responses/BadRequest`, `MethodNotAllowed`, and `InternalError`.

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

Add the schema under `components/schemas`. It describes the envelope, not just the payload:

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

## 4. Examples

Add the request to `examples/requests.http`. In `examples/curl.sh`, use the `expect` helper so the check covers the body:

```bash
body=$(curl -fsS "$BASE_URL/api/items?id=abc123")
expect item "$body" '"data":{'
expect item-id "$body" '"id":"abc123"'
```

## 5. Verify

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

Then confirm the file mapping through the real function builder:

```bash
vercel dev --listen 127.0.0.1:18089
curl 'http://127.0.0.1:18089/api/items?id=abc123'
```

Build the production bundles and confirm the new function exists:

```bash
vercel build --yes
find .vercel/output/functions -name 'items.func'
```

Open `http://localhost:18080/api/docs` and confirm the operation appears and Try it out returns the expected response.

## Rules

- Keep shared code stdlib-only. No new dependency belongs in `api/`, `core/`, `lib/`, or `docs/`.
- Do not add a route outside `/api`.
- Do not add a central route table. The `init` call in the endpoint file is the registration.
- Do not bypass `lib.Handle` for a JSON endpoint. If a handler writes its own JSON, the envelope and error contract stop being uniform.
- Keep the spec and the handler in sync. `docs/extending.md` has the full drift checklist.
