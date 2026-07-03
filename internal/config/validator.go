package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

func (c *Config) portValidator() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("your port number should be between 1 and 65535")
	}
	return nil
}

func (c *Config) healthCheckValidator() error {
	hc := c.HealthCheck
	if hc == nil {
		return nil
	}
	if strings.TrimSpace(hc.Path) == "" {
		return fmt.Errorf("health check path should not be empty")
	}
	if strings.HasPrefix(hc.Path, "/") {
		return fmt.Errorf("health check path should start with /")
	}
	if hc.Interval <= hc.Timeout {
		return fmt.Errorf("interval duration should be greater than timeout interval in health_check")
	}
	if hc.HealthyThreshold < 1 {
		return fmt.Errorf("healthy_threshold should be greater or equal to 1")
	}
	if hc.UnhealthyThreshold < 1 {
		return fmt.Errorf("unhealthy_threshold should be greater or equal to 1")
	}
	if hc.ExpectedStatus < 100 || hc.ExpectedStatus > 599 {
		return fmt.Errorf("expected_status should be a valid HTTP status code")
	}
	return nil
}

func (c *Config) checkDuplicateRoutes() error {
	seen := make(map[string]struct{})

	for _, route := range c.Routes {
		if _, ok := seen[route.Path]; ok {
			return fmt.Errorf("duplicate route path %q", route.Path)
		}

		seen[route.Path] = struct{}{}
	}

	return nil
}

func (c *Config) methodValidation() error {
	for _, route := range c.Routes {
		if len(route.Methods) == 0 {
			return fmt.Errorf("at least one method should be provided for path %q", route.Path)
		}

		for _, m := range route.Methods {
			switch m {
			case GET, POST, PUT, DELETE, PATCH:
				continue
			default:
				return fmt.Errorf("invalid method %q for path %q", m, route.Path)
			}
		}
	}
	return nil
}

func pathValidator(path string) error {
	if path == "" {
		return fmt.Errorf("you can not use empty path")
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("your path must start with /")
	}
	return nil
}

func RateLimitValidator(rl *RateLimit) error {
	if rl == nil {
		return nil
	}
	if rl.Requests <= 0 {
		return fmt.Errorf("requests must be greater than 0")
	}
	duration, err := time.ParseDuration(rl.Per)
	if err != nil {
		return fmt.Errorf("invalid duration %q", rl.Per)
	}
	if duration <= 0 {
		return fmt.Errorf("duration must be greater than 0")
	}
	return nil
}

func timeoutValidator(t *string) error {
	if t == nil {
		return nil
	}
	timeout, err := time.ParseDuration(*t)
	if err != nil {
		return err
	}
	if timeout <= 0 {
		return err
	}
	return nil
}

func backendURLValidator(backendURL string) error {
	u, err := url.ParseRequestURI(backendURL)
	if err != nil {
		return fmt.Errorf("please enter a valid backend_URL")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("error: backend_URL must use http or https scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("error: backend_URL must include a host")
	}
	return nil
}

func (c *Config) Validate() error {
	if err := c.checkDuplicateRoutes(); err != nil {
		return err
	}

	if len(c.Routes) == 0 {
		return fmt.Errorf("at least one route is required")
	}

	if err := c.portValidator(); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	if err := c.methodValidation(); err != nil {
		return err
	}

	if err := c.healthCheckValidator(); err != nil {
		return err
	}

	for i, route := range c.Routes {

		if len(route.BackendURL) == 0 {
			return fmt.Errorf("route[%d]: at least one backend is required", i)
		}

		for i := 0; i < len(route.BackendURL); i++ {
			if err := backendURLValidator(route.BackendURL[i].URL); err != nil {
				return fmt.Errorf("route[%d] invalid backend url: %w", i, err)
			}
		}

		if err := pathValidator(route.Path); err != nil {
			return fmt.Errorf("route[%d] invalid path: %w", i, err)
		}

		if err := RateLimitValidator(route.RateLimit); err != nil {
			return fmt.Errorf("route[%d] invalid rate_limit config: %w", i, err)
		}

		if err := timeoutValidator(route.Timeout); err != nil {
			return fmt.Errorf("route[%d] invalid timeout: %w", i, err)
		}
	}

	return nil
}
