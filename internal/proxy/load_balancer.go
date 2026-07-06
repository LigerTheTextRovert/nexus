package proxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/LigerTheTextRovert/nexus/internal/config"
)

type Backend struct {
	URL     *url.URL
	proxy   *httputil.ReverseProxy
	backend *config.Backend
}

type LoadBalancer struct {
	backends    []Backend
	counter     atomic.Uint64
	path        string
	stripPrefix bool
}

func NewLoadBalancerHandler(backends []config.Backend, path string, stripPrefix bool) (*LoadBalancer, error) {
	backs := make([]Backend, 0, len(backends))

	for i := range backends {
		backend := &backends[i]
		backend.IsHealthy.Store(true)

		targetURL, err := url.Parse(backend.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse backend URL %q: %w", backend.URL, err)
		}
		backs = append(backs, Backend{
			URL:     targetURL,
			proxy:   NewReverseProxy(targetURL),
			backend: backend,
		})
	}

	normalizedPath := path
	if !strings.HasPrefix(normalizedPath, "/") {
		normalizedPath = "/" + normalizedPath
	}

	return &LoadBalancer{
		backends:    backs,
		path:        normalizedPath,
		stripPrefix: stripPrefix,
	}, nil
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend, ok := lb.nextHealthyBackend()
	if !ok {
		slog.Error("no healthy backends available", "path", r.URL.Path, "method", r.Method)
		http.Error(w, "no healthy backend available", http.StatusServiceUnavailable)
		return
	}

	if lb.stripPrefix {
		trimmed := strings.TrimPrefix(r.URL.Path, lb.path)
		if trimmed == "" {
			trimmed = "/"
		} else if !strings.HasPrefix(trimmed, "/") {
			trimmed = "/" + trimmed
		}
		r.URL.Path = trimmed
	}

	backend.proxy.ServeHTTP(w, r)
}

func (lb *LoadBalancer) nextHealthyBackend() (*Backend, bool) {
	if len(lb.backends) == 0 {
		return nil, false
	}

	for range lb.backends {
		// Thread-safe round-robin backend selection.
		current := (lb.counter.Add(1) - 1) % uint64(len(lb.backends))
		backend := &lb.backends[current]
		if backend.isHealthy() {
			return backend, true
		}
	}

	return nil, false
}

func (b *Backend) isHealthy() bool {
	if b.backend == nil {
		return true
	}
	return b.backend.IsHealthy.Load()
}
