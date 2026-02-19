// Package main
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/LigerTheTextRovert/nexus/internal/config"
	"github.com/LigerTheTextRovert/nexus/internal/logging"
	"github.com/LigerTheTextRovert/nexus/internal/proxy"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Initialize the router
	r := chi.NewRouter()
	r.Use(logging.LoggingMiddleware)

	var cfg config.Config

	_, err := config.LoadConfig("configs/config.yml", &cfg)
	if err != nil {
		log.Fatal(err)
	}

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status" : "healthy"}`))
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Gateway is running..."))
	})

	for _, route := range cfg.Routes {
		targetURL, _ := url.Parse(route.BackendURL)
		p := httputil.NewSingleHostReverseProxy(targetURL)

		r.Route(route.Path, func(r chi.Router) {
			r.Handle("/*", proxy.ProxyHandler(p))
		})
	}

	port := cfg.Port
	log.Printf("Starting gateway on port %s...", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
