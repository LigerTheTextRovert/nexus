// Package server stuff(temp package comment).
package server

import (
	"context"
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
	ratelimit "github.com/LigerTheTextRovert/nexus/internal/rate-limit"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	server *http.Server
	config *config.Config

	ctx    context.Context
	cancel context.CancelFunc
}

func New(c *config.Config) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		config: c,
		ctx:    ctx,
		cancel: cancel,
	}

	handler, err := s.routes()
	if err != nil {
		cancel()
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
		if err := s.registerRoute(mux, route); err != nil {
			return nil, err
		}
	}

	return mux, nil
}

func (s *Server) registerRoute(mux chi.Router, route config.Route) error {
	handler, err := proxy.NewLoadBalancerHandler(route.BackendURL, route.Path, route.StripPrefix)
	if err != nil {
		return err
	}

	var manager *ratelimit.RateLimiterManager

	if rl := route.RateLimit; rl != nil {
		per, err := time.ParseDuration(rl.Per)
		if err != nil {
			return fmt.Errorf("invalid rate limit duration for route %q: %w", route.Path, err)
		}

		manager, err = ratelimit.NewManager(
			rl.Requests,
			per,
			time.Minute,
			10*time.Minute,
			s.ctx,
		)
		if err != nil {
			return err
		}
	}

	mux.Route(route.Path, func(r chi.Router) {
		if route.Timeout != nil {
			// we ignore the err cause we validate the
			//config file right after loading it
			duration, _ := time.ParseDuration(*route.Timeout)
			r.Use(middleware.Timeout(duration))
		}

		if manager != nil {
			r.Use(manager.Middleware)
		}

		for _, method := range route.Methods {
			r.Method(string(method), "/*", handler)
		}
	})

	return nil
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
	s.cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}
