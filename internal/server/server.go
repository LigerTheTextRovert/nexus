// Package server stuff(temp package comment).
package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LigerTheTextRovert/nexus/internal/config"
	"github.com/LigerTheTextRovert/nexus/internal/logging"
	"github.com/LigerTheTextRovert/nexus/internal/proxy"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	server *http.Server
	config *config.Config
}

func New(c *config.Config) (*Server, error) {
	s := &Server{
		config: c,
	}

	handler, err := s.routes()
	if err != nil {
		return nil, err
	}

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", c.Port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

func (s *Server) routes() (http.Handler, error) {
	mux := chi.NewRouter()
	mux.Use(logging.LoggingMiddleware)

	mux.Get("/health", healthHandler)
	mux.Get("/", rootHandler)

	for _, route := range s.config.Routes {

		handler, err := proxy.NewLoadBalancerHandler(route.BackendURL, route.Path, route.StripPrefix)
		if err != nil {
			return nil, fmt.Errorf("failed to register a new handler")
		}

		mux.Route(route.Path, func(r chi.Router) {
			r.Handle("/*", handler)
		})
	}

	return mux, nil
}

func (s *Server) Start() error {
	// SIGINT => usually Ctrl + C.
	// SIGTERM => "please terminate" signal sent by Docker, Kubernetes, systemd, etc.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	return s.shutdown()
}

func (s *Server) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}
