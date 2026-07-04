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
	Backends []config.Backend
	client   http.Client
}

func extractBackends(c *config.Config) []config.Backend {
	var backends []config.Backend
	for _, route := range c.Routes {
		backends = append(backends, route.BackendURL...)
	}
	return backends
}

func NewHealthChecker(c *config.Config) HealthChecker {
	return HealthChecker{
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
		wg.Wait()
		go func() {
			defer wg.Done()
			resp, err := hc.client.Get(hc.Backends[i].URL + hc.Config.Path)
			hc.updateHealthStatus(&hc.Backends[i], err)
			defer resp.Body.Close()
		}()
	}
}

func (hc *HealthChecker) updateHealthStatus(backend *config.Backend, err error) {
	if err != nil {
		backend.Failures.Add(1)
		return
	}
	backend.Successes.Add(1)
	if backend.Failures.Load() > int32(hc.Config.UnhealthyThreshold) {
		backend.IsHealthy.Store(false)
	} else {
		backend.IsHealthy.Store(false)
	}
}

func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.Config.Interval)

	for {
		select {
		case <-ticker.C:
			hc.checkBackend()
		case <-ctx.Done():
			return
		}

	}
}
