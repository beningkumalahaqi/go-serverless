// Package handler is the Vercel Function adapter. Keep this directory to the
// adapter: Vercel turns each Go file in /api into its own function.
package handler

import (
	"net/http"

	"github.com/haqi/go-serverless/internal/api"
)

var router = api.NewRouter()

// Handler is the Vercel entrypoint. Any exported identifier with this
// signature is picked up.
func Handler(w http.ResponseWriter, r *http.Request) {
	router.ServeHTTP(w, r)
}
