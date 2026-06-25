package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/LigerTheTextRovert/nexus/internal/config"
)

type Backend struct {
	URL   *url.URL
	proxy *httputil.ReverseProxy
}

type LoadBalancer struct {
	backends    []Backend
	counter     atomic.Uint64
	path        string
	stripPrefix bool
}

func NewLoadBalancerHandler(backends []config.Backend, path string, stripPrefix bool) (*LoadBalancer, error) {
	backs := make([]Backend, 0, len(backends))

	for _, b := range backends {
		targetURL, err := url.Parse(b.Url)
		if err != nil {
			return nil, fmt.Errorf("failed to parse backend URL %q: %w", b, err)
		}
		backs = append(backs, Backend{
			URL:   targetURL,
			proxy: NewReverseProxy(targetURL),
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
	if len(lb.backends) == 0 {
		http.Error(w, "no backend available", http.StatusServiceUnavailable)
		return
	}

	// Theard safe round robin backend selection
	current := (lb.counter.Add(1) - 1) % uint64(len(lb.backends))

	if lb.stripPrefix {
		trimmed := strings.TrimPrefix(r.URL.Path, lb.path)
		if trimmed == "" {
			trimmed = "/"
		} else if !strings.HasPrefix(trimmed, "/") {
			trimmed = "/" + trimmed
		}
		r.URL.Path = trimmed
	}

	lb.backends[current].proxy.ServeHTTP(w, r)
}
