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

type LoadBalancer struct {
	proxies     []*httputil.ReverseProxy
	counter     atomic.Uint64
	path        string
	stripPrefix bool
}

func NewLoadBalancer(
	proxies []*httputil.ReverseProxy,
	path string,
	stripPrefix bool,
) *LoadBalancer {
	return &LoadBalancer{
		proxies:     proxies,
		path:        path,
		stripPrefix: stripPrefix,
	}
}

func NewLoadBalancerHandler(backends []config.Backend, path string, stripPrefix bool) (*LoadBalancer, error) {
	proxies := make([]*httputil.ReverseProxy, 0, len(backends))

	for _, b := range backends {
		targetURL, err := url.Parse(b.Url)
		if err != nil {
			return nil, fmt.Errorf("failed to parse backend URL %q: %w", b, err)
		}
		proxies = append(proxies, NewReverseProxy(targetURL))
	}

	normilizedPath := path
	if !strings.HasPrefix(normilizedPath, "/") {
		normilizedPath = "/" + normilizedPath
	}

	return &LoadBalancer{
		proxies:     proxies,
		path:        normilizedPath,
		stripPrefix: stripPrefix,
	}, nil
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(lb.proxies) == 0 {
		http.Error(w, "no backend available", http.StatusServiceUnavailable)
		return
	}

	// Theard safe round robin backend selection
	current := (lb.counter.Add(1) - 1) % uint64(len(lb.proxies))

	if lb.stripPrefix {
		trimmed := strings.TrimPrefix(r.URL.Path, lb.path)
		if trimmed == "" {
			trimmed = "/"
		} else if !strings.HasPrefix(trimmed, "/") {
			trimmed = "/" + trimmed
		}
		r.URL.Path = trimmed
	}

	lb.proxies[current].ServeHTTP(w, r)
}
