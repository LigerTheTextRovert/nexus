package health

import (
	"context"
	"fmt"
	"log/slog"
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
	if c.HealthCheck == nil {
		return nil
	}

	return &HealthChecker{
		Config:   *c.HealthCheck,
		Backends: extractBackends(c),

		client: http.Client{
			Timeout: c.HealthCheck.Timeout,
		},
	}
}

func (hc *HealthChecker) checkBackend(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(len(hc.Backends))

	for i := range hc.Backends {
		go func(backend *config.Backend) {
			defer wg.Done()

			checkCtx, cancel := context.WithTimeout(ctx, hc.Config.Timeout)
			defer cancel()

			req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, backend.URL+hc.Config.Path, nil)
			if err != nil {
				hc.updateHealthStatus(backend, err)
				return
			}

			resp, err := hc.client.Do(req)
			if err != nil {
				hc.updateHealthStatus(backend, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == hc.Config.ExpectedStatus {
				hc.updateHealthStatus(backend, nil)
			} else {
				hc.updateHealthStatus(backend, fmt.Errorf("unhealthy status: %d", resp.StatusCode))
			}
		}(hc.Backends[i])
	}
	wg.Wait()
}

// The counter represent consecutive results.
func (hc *HealthChecker) updateHealthStatus(backend *config.Backend, err error) {
	if err != nil {
		backend.Successes.Store(0)
		failures := backend.Failures.Add(1)

		if failures >= int32(hc.Config.UnhealthyThreshold) {
			if backend.IsHealthy.Swap(false) {
				slog.Warn("backend became unhealthy", "backend", backend.URL, "err", err)
			}
			backend.Failures.Store(0)
		}

		return
	}

	backend.Failures.Store(0)
	successes := backend.Successes.Add(1)

	if successes >= int32(hc.Config.HealthyThreshold) {
		if !backend.IsHealthy.Swap(true) {
			slog.Info("backend recovered", "backend", backend.URL)
		}
		backend.Successes.Store(0)
	}
}

func (hc *HealthChecker) Start(ctx context.Context) {
	if hc == nil {
		return
	}

	hc.checkBackend(ctx)

	ticker := time.NewTicker(hc.Config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.checkBackend(ctx)
		case <-ctx.Done():
			return
		}
	}
}
