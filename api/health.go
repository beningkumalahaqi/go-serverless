package handler

import (
	"net/http"

	core "haqi/go-serverless/core"
	"haqi/go-serverless/lib"
)

func init() {
	core.Register(http.MethodGet, "/api/health", Health)
}

type healthResponse struct {
	Status string `json:"status"`
}

// Health reports that the endpoint is running.
func Health(w http.ResponseWriter, r *http.Request) {
	lib.Handle(w, r, health, http.MethodGet, http.MethodHead)
}

func health(*http.Request) (any, error) {
	return healthResponse{Status: "ok"}, nil
}
