package config

import (
	"sync/atomic"
	"time"
)

type HTTPMethod = string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
	PATCH  HTTPMethod = "PATCH"
)

type Backend struct {
	URL string `yaml:"url"`

	IsHealthy atomic.Bool

	Failures  atomic.Int32
	Successes atomic.Int32
}

type RateLimit struct {
	Requests int    `yaml:"requests"`
	Per      string `yaml:"per"`
}

type Route struct {
	Path        string       `yaml:"path"`
	Methods     []HTTPMethod `yaml:"methods"`
	BackendURL  []Backend    `yaml:"backends"`
	RateLimit   *RateLimit   `yaml:"rate_limit"`
	Timeout     *string      `yaml:"timeout"`
	StripPrefix bool         `yaml:"strip_prefix"`
}

type HealthCheck struct {
	Path               string        `yaml:"path"`
	Interval           time.Duration `yaml:"interval"`
	Timeout            time.Duration `yaml:"timeout"`
	HealthyThreshold   int           `yaml:"healthy_threshold"`
	UnhealthyThreshold int           `yaml:"unhealthy_threshold"`
	ExpectedStatus     int           `yaml:"expected_status"`
}

type Config struct {
	Port        int          `yaml:"port"`
	HealthCheck *HealthCheck `yaml:"health_check"`
	Routes      []Route      `yaml:"routes"`
}
