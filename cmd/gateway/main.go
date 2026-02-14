// Package main
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/LigerTheTextRovert/nexus/internal/config"
	"github.com/LigerTheTextRovert/nexus/internal/logging"
	"github.com/LigerTheTextRovert/nexus/internal/proxy"
)

func main() {

	var cfg config.Config
	_, err := config.LoadConfig("configs/config.yml", &cfg)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "gateway is running...")
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proxy.ProxyHandler(w, r, &cfg)
		})
		logging.LoggingMiddleware(handler).ServeHTTP(w, r)
	})

	port := cfg.Port
	log.Printf("Starting gateway on port %s...", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
