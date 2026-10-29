package handler

import (
	"net/http"

	core "haqi/go-serverless/core"
	"haqi/go-serverless/docs"
	"haqi/go-serverless/lib"
)

func init() {
	core.Register(http.MethodGet, "/api/docs", Docs)
}

// Docs serves the Swagger UI page for the OpenAPI document. It writes the raw
// HTML page instead of the JSON envelope.
func Docs(w http.ResponseWriter, r *http.Request) {
	if !lib.Allow(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(docs.IndexHTML)
}
