package handler

import (
	"net/http"

	core "haqi/go-serverless/core"
	"haqi/go-serverless/docs"
)

func init() {
	core.Register(http.MethodGet, "/api/docs", Docs)
}

// Docs serves the Swagger UI page for the OpenAPI document.
func Docs(w http.ResponseWriter, r *http.Request) {
	if !core.Allow(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(docs.IndexHTML)
}
