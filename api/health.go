package handler

import (
	"net/http"

	core "haqi/go-serverless/core"
)

func init() {
	core.Register(http.MethodGet, "/api/health", Health)
}

type healthResponse struct {
	Status string `json:"status"`
}

// Health reports that the endpoint is running.
func Health(w http.ResponseWriter, r *http.Request) {
	if !core.Allow(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	core.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}
