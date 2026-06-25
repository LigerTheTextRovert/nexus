package proxy

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/LigerTheTextRovert/nexus/internal/config"
)

func TestProxyConcurrent(t *testing.T) {
	const requestsNumber = 10000
	var received int64
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&received, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backends := []config.Backend{
		{Url: backend.URL},
	}
	proxyHandler, err := NewLoadBalancerHandler(backends, "/api", true)
	if err != nil {
		t.Fatalf("failed to create load balancer: %v", err)
	}

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
