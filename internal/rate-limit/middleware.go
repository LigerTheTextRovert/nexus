// package ratelimit implement a rate limit.
package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Client struct {
	Limiter     *rate.Limiter
	LastRequest time.Time
}

type RateLimiterManager struct {
	clients map[string]*Client
	mu      sync.RWMutex

	request int
	per     time.Duration

	cleanupInterval time.Duration
	idleTimeout     time.Duration
	ctx             context.Context
}

func NewManager(
	requests int,
	per time.Duration,
	cleanupInterval time.Duration,
	idleTimeout time.Duration,
	ctx context.Context,
) (*RateLimiterManager, error) {
	if cleanupInterval <= 0 {
		return nil, fmt.Errorf("Clean up interval should be greater than 0")
	}
	if idleTimeout <= 0 {
		return nil, fmt.Errorf("idle timeout should be greater than 0")
	}

	m := &RateLimiterManager{
		clients:         make(map[string]*Client),
		request:         requests,
		per:             per,
		cleanupInterval: cleanupInterval,
		idleTimeout:     idleTimeout,
	}

	go m.cleanupClients(ctx)
	return m, nil
}

func newLimiter(requests int, per time.Duration) *rate.Limiter {
	limiter := rate.Every(per / time.Duration(requests))
	return rate.NewLimiter(limiter, requests)
}

// We lookup for clients Limiter and return thier own limiter,
// if it's the first time they send a request, we create one for them
func (m *RateLimiterManager) getLimiter(ip string) *rate.Limiter {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, ok := m.clients[ip]
	if !ok {
		newClientLimiter := newLimiter(m.request, m.per)
		m.clients[ip] = &Client{
			Limiter:     newClientLimiter,
			LastRequest: time.Now(),
		}

		return newClientLimiter
	}

	value.LastRequest = time.Now()
	return value.Limiter
}

// Each middleware is a receiver cause we could have different config per route
func (m *RateLimiterManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}

		limiter := m.getLimiter(host)

		if !limiter.Allow() {
			slog.Warn(
				"rate limit exceeded",
				"remote_ip", host,
				"method", r.Method,
				"path", r.URL.Path,
			)
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// In order to prevent memory leaks, we use this clean up function in the background
func (m *RateLimiterManager) cleanupClients(ctx context.Context) {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			for ip, client := range m.clients {
				if time.Since(client.LastRequest) > m.idleTimeout {
					delete(m.clients, ip)
				}
			}
			m.mu.Unlock()

		case <-ctx.Done():
			return
		}
	}
}
