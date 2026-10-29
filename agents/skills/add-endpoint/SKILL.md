---
name: add-endpoint
description: Add or change an HTTP endpoint in this Go API. Use when adding a file under api/, a handler, validation, response types, tests, OpenAPI paths, or example requests.
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

That is the whole route change. Vercel maps `api/items.go` to `/api/items`; `cmd/server` sees it because the package was imported.

Rules:

- Declare `package handler`.
- Export exactly one function with the `http.HandlerFunc` signature. Vercel picks the first exported one in the file.
- Use a handler name that is unique across `api/`. All files compile as one package locally.
- Call `core.Allow` first. A deployed function receives every method sent to its path.
- Reuse `core.WriteJSON` and `core.WriteError` so the JSON and error shapes stay consistent.
- Keep anything that `api/` imports outside `internal/`; Vercel rewrites the module and rejects internal imports.

For JSON bodies, copy the `Echo` handler in `api/echo.go`: cap the body with `http.MaxBytesReader`, decode with `DisallowUnknownFields`, reject trailing content, then validate fields.

Dynamic segments such as `/api/items/{id}` do not populate `r.PathValue` on Vercel. Read the id from the query string or parse `r.URL.Path`.

## 2. Test

Add rows to the table in `core/api_test.go`. The test package imports `api` for its side effects, so the new registration reaches `core.NewRouter`.

```go
{
	name:         "item",
	method:       http.MethodGet,
	path:         "/api/items?id=abc123",
	wantStatus:   http.StatusOK,
	wantContains: `"id":"abc123"`,
},
```

Add the handler to `TestHandlersGuardMethods` so the wrong-method guard is covered without the local router:

```go
{"items", handler.Items, http.MethodPost, "GET, HEAD"},
```

## 3. Spec

Add the path to `docs/openapi.yaml` with every status the handler can return. Reuse `#/components/responses/BadRequest` and `#/components/responses/MethodNotAllowed`.

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

## 4. Examples

Add the request to `examples/requests.http`. In `examples/curl.sh`, use the `expect` helper so the check covers the body:

```bash
body=$(curl -fsS "$BASE_URL/api/items?id=abc123")
expect item "$body" '"id":"abc123"'
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

- Keep shared code stdlib-only. No new dependency belongs in `api/`, `core/`, or `docs/`.
- Do not add a route outside `/api`.
- Do not add a central route table. The `init` call in the endpoint file is the registration.
- Keep the spec and the handler in sync. `docs/extending.md` has the full drift checklist.
