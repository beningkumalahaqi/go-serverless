// Package core holds the route registry and the response helpers shared by the
// endpoint files in api/ and the local server in cmd/server.
package core

import (
	"net/http"
	"sort"
	"strings"
)

type route struct {
	method  string
	pattern string
	handler http.HandlerFunc
}

var (
	routes []route
	taken  = make(map[string]struct{})
)

// Register adds a route. Endpoint files call it from init, so adding a file is
// enough to expose the endpoint from cmd/server. Registration runs during
// package initialization and is not safe for concurrent use.
func Register(method, pattern string, handler http.HandlerFunc) {
	if handler == nil {
		panic("api: nil handler for " + method + " " + pattern)
	}
	key := method + " " + pattern
	if _, ok := taken[key]; ok {
		panic("api: duplicate route " + key)
	}
	taken[key] = struct{}{}
	routes = append(routes, route{method: method, pattern: pattern, handler: handler})
}

// NewRouter returns the handler served by cmd/server. It is built from the
// routes that endpoint files registered during init.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	allowed := make(map[string][]string, len(routes))
	for _, rt := range routes {
		mux.HandleFunc(rt.method+" "+rt.pattern, rt.handler)
		allowed[rt.pattern] = appendMethod(allowed[rt.pattern], rt.method)
		if rt.method == http.MethodGet {
			allowed[rt.pattern] = appendMethod(allowed[rt.pattern], http.MethodHead)
		}
	}
	// A bare path pattern catches wrong methods before the catch-all "/" can
	// turn them into 404s, which is what ServeMux does on its own.
	for pattern, methods := range allowed {
		sort.Strings(methods)
		mux.HandleFunc(pattern, methodNotAllowed(methods))
	}
	mux.HandleFunc("/", notFound)
	return mux
}

func appendMethod(methods []string, method string) []string {
	for _, m := range methods {
		if m == method {
			return methods
		}
	}
	return append(methods, method)
}

func methodNotAllowed(methods []string) http.HandlerFunc {
	allow := strings.Join(methods, ", ")
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", allow)
		WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", r.Method+" is not allowed on "+r.URL.Path)
	}
}

func notFound(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotFound, "not_found", "no route for "+r.Method+" "+r.URL.Path)
}
