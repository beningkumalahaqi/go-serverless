// Package handler is the Vercel Function adapter. Keep this directory to the
// adapter: Vercel turns each Go file in /api into its own function.
package handler

import (
	"net/http"
	"strings"

	"github.com/haqi/go-serverless/internal/api"
)

var router = api.NewRouter()

// Handler is the Vercel entrypoint. Any exported identifier with this
// signature is picked up.
func Handler(w http.ResponseWriter, r *http.Request) {
	// Vercel forwards the rewrite destination, so the original path can arrive
	// as /api/health instead of /health. Strip the prefix when it is there.
	if path, ok := strings.CutPrefix(r.URL.Path, "/api"); ok && (path == "" || path[0] == '/') {
		if path == "" {
			path = "/"
		}
		r.URL.Path = path
	}
	router.ServeHTTP(w, r)
}
