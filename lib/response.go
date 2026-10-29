package lib

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// successBody is the envelope every 2xx JSON response uses.
type successBody struct {
	Data any `json:"data"`
}

// Respond writes data inside the success envelope with the given status.
func Respond(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, successBody{Data: data})
}

// OK writes a 200 response with the success envelope.
func OK(w http.ResponseWriter, data any) {
	Respond(w, http.StatusOK, data)
}

// Created writes a 201 response with the success envelope.
func Created(w http.ResponseWriter, data any) {
	Respond(w, http.StatusCreated, data)
}

// NoContent writes a 204 response with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// writeJSON is the single place that sets the JSON content type and status.
// Endpoints go through Respond, OK, Created, or WriteError instead of calling
// it directly, so no response can skip the shared shape.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}
