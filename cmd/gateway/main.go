// Package main
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

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

	config.LoadConfig("configs/config.yml", &cfg)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(`{"status" : "healthy"}`))
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Gateway is running..."))
	})

	for _, route := range cfg.Routes {
		targetURL, err := url.Parse(route.BackendURL)
		if err != nil {
			log.Fatal("an error occurs during parsing the URL")
		}
		p := httputil.NewSingleHostReverseProxy(targetURL)

		r.Route(route.Path, func(r chi.Router) {
			r.Handle("/*", proxy.ProxyHandler(p, route.Path, route.StripPrefix))
		})
	}

	port := cfg.Port
	log.Printf("Starting gateway on port %d...", port)

	if err := http.ListenAndServe(":"+strconv.Itoa(port), r); err != nil {
		log.Fatal(err)
	}
}
