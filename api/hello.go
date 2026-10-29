package handler

import (
	"net/http"

	core "haqi/go-serverless/core"
)

func init() {
	core.Register(http.MethodGet, "/api/hello", Hello)
}

type helloResponse struct {
	Message string `json:"message"`
}

// Hello greets the optional name query parameter and defaults to World.
func Hello(w http.ResponseWriter, r *http.Request) {
	if !core.Allow(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}
	core.WriteJSON(w, http.StatusOK, helloResponse{Message: "Hello, " + name + "!"})
}
