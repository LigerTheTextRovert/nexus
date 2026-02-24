// Package proxy
package proxy

import (
	"net/http"
	"net/http/httputil"
	"strings"
)

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
