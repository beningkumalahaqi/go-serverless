// Package api holds the host-agnostic HTTP layer. cmd/server calls NewRouter
// and contains no route logic of its own.
package api

import (
	"net/http"
	"strings"
)

type route struct {
	method  string
	pattern string
	handler http.HandlerFunc
}

var routes = []route{
	{http.MethodGet, "/api/health", health},
	{http.MethodGet, "/api/hello", hello},
	{http.MethodPost, "/api/echo", echo},
	{http.MethodGet, "/api/docs", docsPage},
	{http.MethodGet, "/api/openapi.yaml", openapiSpec},
}

// NewRouter returns the handler served by cmd/server.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	allowed := make(map[string][]string, len(routes))
	for _, rt := range routes {
		mux.HandleFunc(rt.method+" "+rt.pattern, rt.handler)
		allowed[rt.pattern] = append(allowed[rt.pattern], rt.method)
		if rt.method == http.MethodGet {
			allowed[rt.pattern] = append(allowed[rt.pattern], http.MethodHead)
		}
	}
	// A bare path pattern catches wrong methods before the catch-all "/" can
	// turn them into 404s, which is what ServeMux does on its own.
	for pattern, methods := range allowed {
		mux.HandleFunc(pattern, methodNotAllowed(methods))
	}
	mux.HandleFunc("/", notFound)
	return mux
}

func methodNotAllowed(methods []string) http.HandlerFunc {
	allow := strings.Join(methods, ", ")
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", allow)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", r.Method+" is not allowed on "+r.URL.Path)
	}
}
