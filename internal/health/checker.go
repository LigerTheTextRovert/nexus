package health

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/LigerTheTextRovert/nexus/internal/config"
)

type HealthChecker struct {
	Config   config.HealthCheck
	Backends []*config.Backend
	client   http.Client
}

func extractBackends(c *config.Config) []*config.Backend {
	// Count total backends first
	total := 0
	for _, route := range c.Routes {
		total += len(route.BackendURL)
	}

	backends := make([]*config.Backend, 0, total)
	for _, route := range c.Routes {
		for i := range route.BackendURL {
			backends = append(backends, &route.BackendURL[i])
		}
	}
	return backends
}

func NewHealthChecker(c *config.Config) *HealthChecker {
	return &HealthChecker{
		Config:   *c.HealthCheck,
		Backends: extractBackends(c),

		client: http.Client{
			Timeout: c.HealthCheck.Timeout,
		},
	}
}

func (hc *HealthChecker) checkBackend() {
	var wg sync.WaitGroup
	wg.Add(len(hc.Backends))
	for i := 0; i < len(hc.Backends); i++ {
		go func(i int) {
			defer wg.Done()
			resp, err := hc.client.Get(hc.Backends[i].URL + hc.Config.Path)
			hc.updateHealthStatus(hc.Backends[i], err)
			defer resp.Body.Close()
		}(i)
	}
	wg.Wait()
}

// The counter represent consecutive results.
func (hc *HealthChecker) updateHealthStatus(backend *config.Backend, err error) {
	if err != nil {
		backend.Successes.Store(0)
		failures := backend.Failures.Add(1)

		if failures >= int32(hc.Config.UnhealthyThreshold) {
			backend.IsHealthy.Store(false)
			backend.Failures.Store(0)
		}

		return
	}

	backend.Failures.Store(0)
	successes := backend.Successes.Add(1)

	if successes >= int32(hc.Config.HealthyThreshold) {
		backend.IsHealthy.Store(true)
		backend.Successes.Store(0)
	}
}

func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.Config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.checkBackend()
		case <-ctx.Done():
			return
		}

	}
}
