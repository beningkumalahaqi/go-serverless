package handler

import (
	"net/http"

	core "haqi/go-serverless/core"
	"haqi/go-serverless/docs"
	"haqi/go-serverless/lib"
)

func init() {
	core.Register(http.MethodGet, "/api/openapi.yaml", OpenAPI)
}

// OpenAPI serves the embedded OpenAPI document as raw YAML instead of the JSON
// envelope.
func OpenAPI(w http.ResponseWriter, r *http.Request) {
	if !lib.Allow(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Write(docs.OpenAPI)
}
