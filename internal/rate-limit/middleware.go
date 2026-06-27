// package ratelimit implement a rate limit.
package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Client struct {
	Limiter     *rate.Limiter
	LastRequest time.Time
}

var (
	Clients = make(map[string]*Client)
	mu      sync.Mutex
)

func NewLimiter(requsts int, per time.Duration) *rate.Limiter {
	limiter := rate.Every(per / time.Duration(requsts))
	return rate.NewLimiter(limiter, requsts)
}

func GetLimiter(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	value, ok := Clients[ip]
	if !ok {
		newClientLimiter := NewLimiter(100, 1)
		Clients[ip] = &Client{
			Limiter:     newClientLimiter,
			LastRequest: time.Now(),
		}

		return newClientLimiter
	}

	value.LastRequest = time.Now()
	return value.Limiter
}

func RateLimiterMiddlewave(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		limiter := GetLimiter(ip)

		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// In order to prevent memory leaks, we use this clean up function in the background
func CleanupClients() {
	ticker := time.NewTicker(time.Minute)
	for {
		<-ticker.C
		mu.Lock()
		for ip, client := range Clients {
			if time.Since(client.LastRequest) > 10*time.Minute {
				delete(Clients, ip)
			}
		}
		mu.Unlock()
	}
}
