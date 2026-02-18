// Package proxy
package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type App struct {
	TargetURL string
}

func (a *App) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	targetURL, err := url.Parse(a.TargetURL)
	if err != nil {
		http.Error(w, "Proxy Error", http.StatusInternalServerError)
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = &http.Transport{ResponseHeaderTimeout: 10 * time.Second}

	r.Host = targetURL.Host
	proxy.ServeHTTP(w, r)
}
