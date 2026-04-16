// Package server stuff(temp package comment).
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
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

func New(c *config.Config) *Server {
	s := &Server{
		config: c,
	}

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", c.Port),
		Handler:      s.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) routes() http.Handler {
	mux := chi.NewRouter()
	mux.Use(logging.LoggingMiddleware)

	for _, route := range s.config.Routes {
		targetURL, err := url.Parse(route.BackendURL)
		if err != nil {
			log.Fatal("an error occurs during parsing the URL")
		}
		p := httputil.NewSingleHostReverseProxy(targetURL)

		mux.Route(route.Path, func(r chi.Router) {
			r.Handle("/*", proxy.ProxyHandler(p, route.Path, route.StripPrefix))
		})
	}

	return mux
}

func (s *Server) Start() error {
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
