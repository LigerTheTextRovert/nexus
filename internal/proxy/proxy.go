// Package proxy
package proxy

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

func NewReverseProxy(target *url.URL) *httputil.ReverseProxy {

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error(
			"upstream request failed",
			"backend", target,
			"method", r.Method,
			"path", r.URL.Path,
			"err", err,
		)

		switch {
		case errors.Is(err, context.DeadlineExceeded):
			http.Error(w, "upstream request timeout", http.StatusGatewayTimeout)
			return
		default:
			http.Error(w, "service unavailable", http.StatusBadGateway)
			return
		}
	}

	return proxy
}

func ProxyHandler(proxy *httputil.ReverseProxy, path string, stripPrefix bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stripPrefix {
			// We normalize the path if the incoming path doesn't start with /
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			trimmed := strings.TrimPrefix(r.URL.Path, path)
			if trimmed == "" {
				trimmed = "/"
			} else if !strings.HasPrefix(trimmed, "/") {
				trimmed = "/" + trimmed
			}
			r.URL.Path = trimmed
		}
		proxy.ServeHTTP(w, r)
	})
}
