package proxy

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/LigerTheTextRovert/nexus/internal/config"
)

func TestLoadBalancer_NoBackends(t *testing.T) {
	lb := &LoadBalancer{backends: []Backend{}}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()
	lb.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
}

func TestNewLoadBalancerHandler_InvalidURL(t *testing.T) {
	_, err := NewLoadBalancerHandler([]config.Backend{{URL: "://bad-url"}}, "/api", false)
	if err == nil {
		t.Error("expected error for invalid backend URL, got nil")
	}
}

// TestLoadBalancer_RoundRobin verifies that requests are distributed evenly
// across backends in a sequential round-robin fashion.
func TestLoadBalancer_RoundRobin(t *testing.T) {
	var hits [2]atomic.Int64

	backend0 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[0].Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend0.Close()

	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[1].Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend1.Close()

	lb, err := NewLoadBalancerHandler([]config.Backend{
		{URL: backend0.URL},
		{URL: backend1.URL},
	}, "/api", false)
	if err != nil {
		t.Fatalf("NewLoadBalancerHandler: %v", err)
	}

	const total = 10
	for range total {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rr := httptest.NewRecorder()
		lb.ServeHTTP(rr, req)
	}

	if hits[0].Load() != 5 || hits[1].Load() != 5 {
		t.Errorf("uneven distribution: backend0=%d backend1=%d (expected 5 each)",
			hits[0].Load(), hits[1].Load())
	}
}

// TestLoadBalancer_RoundRobin_Concurrent verifies that the atomic counter
// stays race-free and the total hit count is exact under concurrent load.
// Complement to proxy_concurrent_test.go which uses a single backend.
func TestLoadBalancer_RoundRobin_Concurrent(t *testing.T) {
	var hits [2]atomic.Int64

	backend0 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[0].Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend0.Close()

	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[1].Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend1.Close()

	lb, err := NewLoadBalancerHandler([]config.Backend{
		{URL: backend0.URL},
		{URL: backend1.URL},
	}, "/api", false)
	if err != nil {
		t.Fatalf("NewLoadBalancerHandler: %v", err)
	}

	const total = 1000
	var wg sync.WaitGroup
	wg.Add(total)

	for range total {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			rr := httptest.NewRecorder()
			lb.ServeHTTP(rr, req)
		}()
	}

	wg.Wait()

	if got := hits[0].Load() + hits[1].Load(); got != total {
		t.Errorf("total hit count mismatch: got %d, expected %d", got, total)
	}
}

func TestLoadBalancer_SkipsUnhealthyBackends(t *testing.T) {
	var unhealthyHits atomic.Int64
	var healthyHits atomic.Int64

	unhealthyBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		unhealthyHits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer unhealthyBackend.Close()

	healthyBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		healthyHits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer healthyBackend.Close()

	backends := []config.Backend{
		{URL: unhealthyBackend.URL},
		{URL: healthyBackend.URL},
	}
	lb, err := NewLoadBalancerHandler(backends, "/api", false)
	if err != nil {
		t.Fatalf("NewLoadBalancerHandler: %v", err)
	}
	backends[0].IsHealthy.Store(false)

	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rr := httptest.NewRecorder()
		lb.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
		}
	}

	if unhealthyHits.Load() != 0 {
		t.Errorf("unhealthy backend received %d requests", unhealthyHits.Load())
	}
	if healthyHits.Load() != 5 {
		t.Errorf("healthy backend received %d requests, expected 5", healthyHits.Load())
	}
}

func TestLoadBalancer_AllBackendsUnhealthy(t *testing.T) {
	backend0 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend0.Close()

	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend1.Close()

	backends := []config.Backend{
		{URL: backend0.URL},
		{URL: backend1.URL},
	}
	lb, err := NewLoadBalancerHandler(backends, "/api", false)
	if err != nil {
		t.Fatalf("NewLoadBalancerHandler: %v", err)
	}
	for i := range backends {
		backends[i].IsHealthy.Store(false)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()
	lb.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
}
