package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
)

func TestProxy(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		stripPrefix bool
		reqPath     string
		wantPath    string
	}{
		{
			name:        "no_strip_prefix",
			path:        "/api",
			stripPrefix: false,
			reqPath:     "/api/users",
			wantPath:    "/api/users",
		},
		{
			name:        "strip_prefix",
			path:        "/api",
			stripPrefix: true,
			reqPath:     "/api/users",
			wantPath:    "/users",
		},
		{
			name:        "strip_prefix_root",
			path:        "/api",
			stripPrefix: true,
			reqPath:     "/api",
			wantPath:    "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string

			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			target, _ := url.Parse(backend.URL)
			proxy := httputil.NewSingleHostReverseProxy(target)

			handler := ProxyHandler(proxy, tt.path, tt.stripPrefix)

			req := httptest.NewRequest(http.MethodGet, tt.reqPath, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if receivedPath != tt.wantPath {
				t.Fatalf("expected path %s, got %s", tt.wantPath, receivedPath)
			}
		})
	}
}
