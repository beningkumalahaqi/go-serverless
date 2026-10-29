// Package docs embeds the OpenAPI spec and the Swagger UI page so the binary
// serves its own documentation on any host.
package docs

import _ "embed"

//go:embed openapi.yaml
var OpenAPI []byte

//go:embed index.html
var IndexHTML []byte
