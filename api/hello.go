package handler

import (
	"net/http"

	core "haqi/go-serverless/core"
	"haqi/go-serverless/lib"
)

func init() {
	core.Register(http.MethodGet, "/api/hello", Hello)
}

type helloResponse struct {
	Message string `json:"message"`
}

// Hello greets the optional name query parameter and defaults to World.
func Hello(w http.ResponseWriter, r *http.Request) {
	lib.Handle(w, r, hello, http.MethodGet, http.MethodHead)
}

func hello(r *http.Request) (any, error) {
	name := lib.Query(r, "name", "World")
	return helloResponse{Message: "Hello, " + name + "!"}, nil
}
