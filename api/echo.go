package handler

import (
	"net/http"
	"strings"
	"time"

	core "haqi/go-serverless/core"
	"haqi/go-serverless/lib"
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
	lib.Handle(w, r, echo, http.MethodPost)
}

func echo(r *http.Request) (any, error) {
	var req echoRequest
	if err := lib.DecodeJSON(r, &req, maxEchoBytes); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Message) == "" {
		return nil, lib.BadRequest("message is required")
	}
	return echoResponse{Message: req.Message, ReceivedAt: time.Now().UTC()}, nil
}
