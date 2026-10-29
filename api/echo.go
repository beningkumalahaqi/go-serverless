package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	core "haqi/go-serverless/core"
)

const maxEchoBytes = 1 << 10

func init() {
	core.Register(http.MethodPost, "/api/echo", Echo)
}

type echoRequest struct {
	Message string `json:"message"`
}

type echoResponse struct {
	Message    string    `json:"message"`
	ReceivedAt time.Time `json:"receivedAt"`
}

// Echo returns the JSON message it receives with a UTC timestamp.
func Echo(w http.ResponseWriter, r *http.Request) {
	if !core.Allow(w, r, http.MethodPost) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxEchoBytes)
	var req echoRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		switch {
		case errors.As(err, &tooLarge):
			core.WriteError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "body must be at most 1 KB")
		case errors.Is(err, io.EOF):
			core.WriteError(w, http.StatusBadRequest, "invalid_request", "body is required")
		default:
			core.WriteError(w, http.StatusBadRequest, "invalid_request", "body must be valid JSON with a message field")
		}
		return
	}
	// Decode stops after the first value, so a second one would otherwise pass.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		core.WriteError(w, http.StatusBadRequest, "invalid_request", "body must contain exactly one JSON object")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		core.WriteError(w, http.StatusBadRequest, "invalid_request", "message is required")
		return
	}

	core.WriteJSON(w, http.StatusOK, echoResponse{Message: req.Message, ReceivedAt: time.Now().UTC()})
}
