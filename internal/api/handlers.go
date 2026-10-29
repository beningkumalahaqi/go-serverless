package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/haqi/go-serverless/docs"
)

const maxEchoBytes = 1 << 10

type healthResponse struct {
	Status string `json:"status"`
}

type helloResponse struct {
	Message string `json:"message"`
}

type echoRequest struct {
	Message string `json:"message"`
}

type echoResponse struct {
	Message    string    `json:"message"`
	ReceivedAt time.Time `json:"receivedAt"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func hello(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}
	writeJSON(w, http.StatusOK, helloResponse{Message: "Hello, " + name + "!"})
}

func echo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxEchoBytes)
	var req echoRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		switch {
		case errors.As(err, &tooLarge):
			writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "body must be at most 1 KB")
		case errors.Is(err, io.EOF):
			writeError(w, http.StatusBadRequest, "invalid_request", "body is required")
		default:
			writeError(w, http.StatusBadRequest, "invalid_request", "body must be valid JSON with a message field")
		}
		return
	}
	// Decode stops after the first value, so a second one would otherwise pass.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_request", "body must contain exactly one JSON object")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "message is required")
		return
	}

	writeJSON(w, http.StatusOK, echoResponse{Message: req.Message, ReceivedAt: time.Now().UTC()})
}

func docsPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(docs.IndexHTML)
}

func openapiSpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Write(docs.OpenAPI)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "no route for "+r.Method+" "+r.URL.Path)
}
