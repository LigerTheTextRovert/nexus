package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
)

func TestProxyConcurrent(t *testing.T) {
	const requestsNumber = 10000
	var received int64
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&received, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	target, _ := url.Parse(backend.URL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxyHandler := ProxyHandler(proxy, "/api", true)

	var wg sync.WaitGroup
	wg.Add(requestsNumber)

	for range requestsNumber {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
			rr := httptest.NewRecorder()
			proxyHandler.ServeHTTP(rr, req)
		}()
	}

	wg.Wait()

	if received != requestsNumber {
		t.Fatalf("error: expected %d requests, got %d", requestsNumber, received)
	}

}
