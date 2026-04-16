// Package main
package main

import (
	"log"
	"net/http"

	"github.com/LigerTheTextRovert/nexus/internal/config"
	"github.com/LigerTheTextRovert/nexus/internal/logging"
	"github.com/LigerTheTextRovert/nexus/internal/server"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Initialize the router
	r := chi.NewRouter()
	r.Use(logging.LoggingMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(`{"status" : "healthy"}`))
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Gateway is running..."))
	})

	if err := run(); err != nil {
		log.Fatal(err)
	}

}

func run() error {
	var cfg config.Config

	config.LoadConfig("configs/config.yml", &cfg)
	cfg.Validate()

	server := server.New(&cfg)
	return server.Start()
}
