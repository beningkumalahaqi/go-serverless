package lib

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sort"
	"strings"
)

// Handler is the business logic of an endpoint. It never touches the
// ResponseWriter: it returns the value for the success envelope, or an error
// that Handle turns into the shared error shape.
type Handler func(r *http.Request) (any, error)

// Handle adapts a Handler to the standard http.HandlerFunc signature. It
// checks the method, recovers panics, maps errors, and wraps successful values
// in the data envelope with a 200 status.
func Handle(w http.ResponseWriter, r *http.Request, h Handler, methods ...string) {
	HandleStatus(w, r, http.StatusOK, h, methods...)
}

// HandleStatus is Handle with a different 2xx success status. A 204 writes no
// body; every other status uses the data envelope.
func HandleStatus(w http.ResponseWriter, r *http.Request, status int, h Handler, methods ...string) {
	if len(methods) == 0 {
		panic("lib: Handle needs at least one method")
	}
	if !Allow(w, r, methods...) {
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("panic recovered",
				"method", r.Method,
				"path", r.URL.Path,
				"panic", recovered,
				"stack", string(debug.Stack()),
			)
			WriteError(w, r, Internal(fmt.Errorf("panic: %v", recovered)))
		}
	}()

	data, err := h(r)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	if status == http.StatusNoContent {
		NoContent(w)
		return
	}
	Respond(w, status, data)
}

// Allow checks the request method against the methods an endpoint accepts.
// When the method is not allowed it writes a 405 response and returns false.
// A deployed function receives every method sent to its path, so handlers
// behind Handle are checked on every host.
func Allow(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	allowed := append([]string(nil), methods...)
	sort.Strings(allowed)
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	WriteError(w, r, NewError(http.StatusMethodNotAllowed, CodeMethodNotAllowed,
		r.Method+" is not allowed on "+r.URL.Path))
	return false
}
