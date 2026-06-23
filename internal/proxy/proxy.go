// Package proxy
package proxy

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
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
