// Package proxy
package proxy

import (
	"net/http"
	"net/http/httputil"
)

func ProxyHandler(proxy *httputil.ReverseProxy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})
}
