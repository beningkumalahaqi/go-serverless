package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "haqi/go-serverless/api"
	"haqi/go-serverless/core"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	srv := &http.Server{
		Addr:              addr,
		Handler:           core.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("listening", "url", "http://localhost"+addr, "docs", "http://localhost"+addr+"/api/docs")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
