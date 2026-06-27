// package ratelimit implement a rate limit.
package ratelimit

import (
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

type Manager struct {
	Clients map[string]*Client
	mu      sync.RWMutex

	Request int
	Per     time.Duration
}

func NewManager(requests int, per time.Duration) *Manager {
	m := &Manager{
		Clients: make(map[string]*Client),
		Request: requests,
		Per:     per,
	}

	go m.CleanupClients()
	return m
}

func NewLimiter(requsts int, per time.Duration) *rate.Limiter {
	limiter := rate.Every(per / time.Duration(requsts))
	return rate.NewLimiter(limiter, requsts)
}

// We lookup for clients Limiter and return thier own limiter,
// if it's the first time they send a request, we create one for them
func (m *Manager) GetLimiter(ip string) *rate.Limiter {
	m.mu.RLock()
	value, ok := m.Clients[ip]
	defer m.mu.RUnlock()

	if !ok {
		newClientLimiter := NewLimiter(100, 1)
		m.Clients[ip] = &Client{
			Limiter:     newClientLimiter,
			LastRequest: time.Now(),
		}

		return newClientLimiter
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	value.LastRequest = time.Now()
	return value.Limiter
}

func (m *Manager) RateLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}

		limiter := m.GetLimiter(host)

		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// In order to prevent memory leaks, we use this clean up function in the background
func (m *Manager) CleanupClients() {
	ticker := time.NewTicker(time.Minute)
	for {
		<-ticker.C
		m.mu.Lock()
		for ip, client := range m.Clients {
			if time.Since(client.LastRequest) > 10*time.Minute {
				delete(m.Clients, ip)
			}
		}
		m.mu.Unlock()
	}
}
