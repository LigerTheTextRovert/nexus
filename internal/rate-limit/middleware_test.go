package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestManager is a helper to reduce boilerplate in every test case.
func newTestManager(t *testing.T, requests int, per, cleanupInterval, idleTimeout time.Duration) *RateLimiterManager {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	mgr, err := NewManager(requests, per, cleanupInterval, idleTimeout, ctx)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return mgr
}

func TestNewManager_InvalidParams(t *testing.T) {
	tests := []struct {
		name            string
		cleanupInterval time.Duration
		idleTimeout     time.Duration
		wantErr         bool
	}{
		{"valid", time.Minute, 5 * time.Minute, false},
		{"zero_cleanup_interval", 0, 5 * time.Minute, true},
		{"negative_cleanup_interval", -time.Second, 5 * time.Minute, true},
		{"zero_idle_timeout", time.Minute, 0, true},
		{"negative_idle_timeout", time.Minute, -time.Second, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			_, err := NewManager(10, time.Second, tt.cleanupInterval, tt.idleTimeout, ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewManager() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestMiddleware_AllowsRequestWithinLimit(t *testing.T) {
	mgr := newTestManager(t, 5, time.Second, time.Minute, 5*time.Minute)

	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rr := httptest.NewRecorder()

	mgr.Middleware(backend).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestMiddleware_BlocksWhenBucketExhausted(t *testing.T) {
	const limit = 3
	mgr := newTestManager(t, limit, time.Hour, time.Minute, 5*time.Minute)

	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := mgr.Middleware(backend)

	// Exhaust all tokens.
	for i := range limit {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected %d, got %d", i+1, http.StatusOK, rr.Code)
		}
	}

	// Next request must be rejected.
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected %d, got %d", http.StatusTooManyRequests, rr.Code)
	}
}

// TestMiddleware_IndependentBucketsPerIP verifies that exhausting one IP's bucket
// does not affect another IP.
func TestMiddleware_IndependentBucketsPerIP(t *testing.T) {
	const limit = 2
	mgr := newTestManager(t, limit, time.Hour, time.Minute, 5*time.Minute)

	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := mgr.Middleware(backend)

	sendN := func(ip string, n int) int {
		var lastStatus int
		for range n {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.RemoteAddr = ip + ":1234"
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			lastStatus = rr.Code
		}
		return lastStatus
	}

	// Exhaust IP A's bucket.
	sendN("1.1.1.1", limit)

	if got := sendN("1.1.1.1", 1); got != http.StatusTooManyRequests {
		t.Errorf("IP A: expected %d, got %d", http.StatusTooManyRequests, got)
	}

	// IP B should be completely unaffected.
	if got := sendN("2.2.2.2", 1); got != http.StatusOK {
		t.Errorf("IP B: expected %d, got %d", http.StatusOK, got)
	}
}

// TestMiddleware_MalformedRemoteAddr verifies that when RemoteAddr has no port,
// the full address is used as the client key without crashing.
func TestMiddleware_MalformedRemoteAddr(t *testing.T) {
	mgr := newTestManager(t, 5, time.Second, time.Minute, 5*time.Minute)

	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.1" // no port
	rr := httptest.NewRecorder()

	mgr.Middleware(backend).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestGetLimiter_ReturnsSameInstanceForSameIP verifies that repeat calls return the
// same limiter (same bucket) rather than resetting it each time.
func TestGetLimiter_ReturnsSameInstanceForSameIP(t *testing.T) {
	mgr := newTestManager(t, 10, time.Second, time.Minute, 5*time.Minute)

	l1 := mgr.getLimiter("10.0.0.1")
	l2 := mgr.getLimiter("10.0.0.1")

	if l1 != l2 {
		t.Error("expected same limiter instance for the same IP, got different pointers")
	}
}

// TestCleanupClients verifies that idle clients are evicted from the internal map.
func TestCleanupClients(t *testing.T) {
	mgr := newTestManager(t, 10, time.Second, 25*time.Millisecond, 50*time.Millisecond)

	// Register a client.
	mgr.getLimiter("172.16.0.1")

	mgr.mu.RLock()
	n := len(mgr.clients)
	mgr.mu.RUnlock()

	if n != 1 {
		t.Fatalf("expected 1 client before cleanup, got %d", n)
	}

	// Wait long enough for the idle timeout and at least one cleanup tick.
	time.Sleep(200 * time.Millisecond)

	mgr.mu.RLock()
	n = len(mgr.clients)
	mgr.mu.RUnlock()

	if n != 0 {
		t.Errorf("expected 0 clients after cleanup, got %d", n)
	}
}
