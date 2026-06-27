package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LigerTheTextRovert/nexus/internal/config"
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

			backends := []config.Backend{
				{URL: backend.URL},
			}
			handler, err := NewLoadBalancerHandler(backends, tt.path, tt.stripPrefix)
			if err != nil {
				t.Fatalf("failed to create load balancer: %v", err)
			}
			req := httptest.NewRequest(http.MethodGet, tt.reqPath, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if receivedPath != tt.wantPath {
				t.Fatalf("expected path %s, got %s", tt.wantPath, receivedPath)
			}
		})
	}
}
