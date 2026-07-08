<p align="center">
  <img src="assets/logo.png" alt="Nexus Logo" width="180">
</p>

<h1 align="center">Nexus API Gateway</h1>

<p align="center">
    A high-performance, concurrent, and lightweight HTTP API Gateway and Reverse Proxy written in Go. Nexus sits in front of your microservices, managing traffic routing, path rewriting, rate limiting, route timeouts, active backend health-checks, and structured logging.
</p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go Version"></a>
  <a href="#testing"><img src="https://img.shields.io/badge/Tests-Passing-brightgreen?style=flat-square" alt="Tests Status"></a>
  <a href="#concurrency-and-thread-safety-deep-dive"><img src="https://img.shields.io/badge/Concurrency-Lock--Free%20%2F%20Atomic-blue?style=flat-square" alt="Concurrency Mode"></a>
</p>

---

## 📖 Navigation

- [Key Architectural Design](#-key-architectural-design)
  - [Project Layout](#project-layout)
  - [Request Lifecycle](#request-lifecycle)
- [⚡ Concurrency & Thread-Safety Deep-Dive](#-concurrency--thread-safety-deep-dive)
  - [1. Lock-Free Round-Robin Selection](#1-lock-free-round-robin-selection)
  - [2. Concurrent Active Health Probing](#2-concurrent-active-health-probing)
  - [3. Atomic State Management & Transitions](#3-atomic-state-management--transitions)
  - [4. Safe Rate Limiting & Memory Leak Prevention](#4-safe-rate-limiting--memory-leak-prevention)
- [✨ Core Features](#-core-features)
- [🛠️ Configuration & Strict Startup Validation](#%EF%B8%8F-configuration--strict-startup-validation)
  - [Configuration Schema (`configs/config.yml`)](#configuration-schema-configsconfigyml)
  - [Strict Validation Matrix](#strict-validation-matrix)
- [📝 Observability & Structured Logging (`slog`)](#-observability--structured-logging-slog)
  - [Example Log Outputs](#example-log-outputs)
- [🚨 Error Semantics & Troubleshooting](#-error-semantics--troubleshooting)
- [🚀 Getting Started & Local Development](#-getting-started--local-development)
  - [Prerequisites](#prerequisites)
  - [Running the Gateway & Mock Services](#running-the-gateway--mock-services)
  - [Running Tests & Race Detection](#running-tests--race-detection)
- [🧩 Third-Party Dependencies Philosophy](#-third-party-dependencies-philosophy)
- [🗺️ Future Roadmap](#%EF%B8%8F-future-roadmap)

---

## 🏛️ Key Architectural Design

Nexus is built with the principle of **low-overhead, production-ready routing**. It coordinates several independent modules (configuration validator, active health checker, rate limiter, and load-balancer reverse proxy) utilizing Go's highly optimized standard library scheduler.

```
Client Request
       │
       ▼
 [chi router] ────► Structured JSON Logging (slog)
       │
       ├─► GET /health  ──────► {"status": "healthy"}
       ├─► GET /        ──────► "Gateway is running..."
       │
       ├─► /api/users/* ──────► [Timeout: 10s] ──► [Rate Limit: 100 req/m per IP]
       │                            │
       │                            ▼
       │                      [Health-Aware Round-Robin Load Balancer]
       │                            │
       │                            ├─► Healthy Upstream: http://localhost:8081 (strip_prefix)
       │                            └─► Unhealthy Upstream: http://localhost:8083 (Skipped!)
       │
       └─► /api/orders/* ─────► [Timeout: 5s] ──► [Rate Limit: 50 req/m per IP]
                                    │
                                    ▼
                              [Health-Aware Round-Robin Load Balancer]
                                    │
                                    └─► Healthy Upstream: http://localhost:8082 (strip_prefix)

─────────────────────────────────────────────────────────────────────────────
Background Routines:
  [Health Checker] ──► Every 5s ──► GET {backend}/health
       │
       ├─► 2 Consecutive Successes  ──► Mark Backend HEALTHY (Lock-free atomic swap)
       └─► 3 Consecutive Failures   ──► Mark Backend UNHEALTHY (Lock-free atomic swap)
```

### Project Layout

Nexus follows the [Standard Go Project Layout](https://github.com/golang-standards/project-layout) to separate concerns and enforce clean package boundaries:

```
nexus/
  ├── cmd/
  │    ├── gateway/             # Gateway entrypoint (config loading, validation, server start)
  │    ├── users-service/       # Mock Upstream Service (Port 8081)
  │    └── orders-service/      # Mock Upstream Service (Port 8082)
  ├── configs/
  │    └── config.yml           # Runtime YAML configuration file
  ├── internal/
  │    ├── config/              # Configuration schemas, parser, and strict validator
  │    ├── health/              # Active background HTTP health-checker engine
  │    ├── logging/             # Structured JSON logging HTTP middleware
  │    ├── proxy/               # Reverse proxy transport, error handling, and Load Balancer
  │    ├── rate-limit/          # Thread-safe client-IP token-bucket rate limiter with cleanup
  │    └── server/              # Graceful HTTP server setup and routing wiring
  ├── pkg/utils/                # Reusable public utility helpers
  ├── Makefile                  # Convenient tooling & orchestration commands
  └── README.md                 # Technical documentation
```

### Request Lifecycle

1. **Ingress**: A client request is picked up by the `chi` router.
2. **Logging Middleware**: The request's metadata (IP, Method, Path, User-Agent) is captured and a custom `ResponseWriter` is injected to intercept the status code and response size.
3. **Timeout Middleware**: If a route timeout is configured, a context with a timeout deadline is injected.
4. **Rate Limiting**: The client IP is extracted. If the IP has exceeded its per-route bucket limits, a `429 Too Many Requests` is returned and a warning is logged.
5. **Load Balancing**: The Load Balancer queries the next backend URL. It skips any backend that has been marked unhealthy. If no healthy backends are available, a `503 Service Unavailable` is returned.
6. **Reverse Proxying**: The Load Balancer applies path rewriting (e.g. stripping prefixes if configured), updates the request target, and forwards the request to the backend using `httputil.ReverseProxy` over a custom, tuned `http.Transport`.
7. **Egress & Log Completion**: The response is streamed back to the client, and the Logging Middleware outputs a single structured JSON log containing the overall response status, duration, and body bytes.

---

## ⚡ Concurrency & Thread-Safety Deep-Dive

Nexus is designed to handle thousands of concurrent requests with minimal mutex contention. By leveraging atomic primitives and optimized lock granularity, it guarantees strict thread safety under massive load.

### 1. Lock-Free Round-Robin Selection
Traditional load balancers often wrap backend lists and selection indices in heavy mutual exclusion locks (`sync.Mutex`), which causes execution bottlenecks when handling thousands of requests concurrently. 

Nexus implements a **lock-free round-robin algorithm** using Go's `sync/atomic` package:
```go
func (lb *LoadBalancer) nextHealthyBackend() (*Backend, bool) {
	if len(lb.backends) == 0 {
		return nil, false
	}

	for range lb.backends {
		// Thread-safe round-robin selection without mutex locking
		current := (lb.counter.Add(1) - 1) % uint64(len(lb.backends))
		backend := &lb.backends[current]
		if backend.isHealthy() {
			return backend, true
		}
	}

	return nil, false
}
```
* **Mechanism**: `atomic.Uint64.Add(1)` is a CPU-level atomic instruction that avoids thread context switching or lock overhead. The modulo operation wraps the index around the backends array safely. If a selected backend is unhealthy, the loop automatically increments and checks the next available backend.

### 2. Concurrent Active Health Probing
Rather than probing backends sequentially (which slows down checking intervals as upstream backends grow), the health checker executes queries in parallel using lightweight goroutines and synchronizes them using a `sync.WaitGroup`:

```go
func (hc *HealthChecker) checkBackend(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(len(hc.Backends))

	for i := range hc.Backends {
		go func(backend *config.Backend) {
			defer wg.Done()
			// Per-backend timeout context ensures one slow backend doesn't stall others
			checkCtx, cancel := context.WithTimeout(ctx, hc.Config.Timeout)
			defer cancel()

			req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, backend.URL+hc.Config.Path, nil)
			// ... HTTP execution and update status
		}(hc.Backends[i])
	}
	wg.Wait()
}
```
* **Mechanism**: Each probe runs concurrently. A per-backend context timeout ensures that a hanging upstream backend does not delay the health check execution of other healthy backends.

### 3. Atomic State Management & Transitions
To bridge the background health-check goroutines and the main request-handling routing goroutines, Nexus uses lock-free atomic swaps.

Inside the health-check handler:
```go
func (hc *HealthChecker) updateHealthStatus(backend *config.Backend, err error) {
	if err != nil {
		backend.Successes.Store(0)
		failures := backend.Failures.Add(1)

		if failures >= int32(hc.Config.UnhealthyThreshold) {
			// Swap is atomic; returns true if the previous state was indeed healthy
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
		// Swap is atomic; returns false if the previous state was unhealthy
		if !backend.IsHealthy.Swap(true) {
			slog.Info("backend recovered", "backend", backend.URL)
		}
		backend.Successes.Store(0)
	}
}
```
* **Mechanism**: By using `backend.IsHealthy.Swap(val)`, we ensure that state transitions are executed atomically and that log lines regarding a state change are emitted **exactly once** upon crossing a success/failure threshold. This avoids log-spamming during intermittent connection issues.

### 4. Safe Rate Limiting & Memory Leak Prevention
Each route can define its own rate-limiting criteria. Nexus maintains a map of IP addresses to their respective token-bucket limiters.

To prevent memory leaks from one-off or short-lived clients (such as port scanners or transient visitors), the `RateLimiterManager` implements a synchronized background sweep routine:
```go
func (m *RateLimiterManager) cleanupClients(ctx context.Context) {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			for ip, client := range m.clients {
				// Safely delete client buckets that have been inactive past the idleTimeout
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
```
* **Thread-Safety**: Readers use `RWMutex.RLock()` / `RUnlock()` during active API calls to look up limiters, allowing highly concurrent lookups. The cleanup worker acquires a write `Lock()` on a slow tick (e.g. every minute) to safely purge stale, idle entries.

---

## ✨ Core Features

* **Declarative Routing Configuration**: Route prefixes, allowed methods, upstreams, timeouts, and limits are declared cleanly in a single, simple YAML file.
* **Health-Aware Load Balancing**: Multi-backend support with thread-safe, zero-allocation round-robin routing that intelligently skips downed servers.
* **Active Background Probing**: Configurable health endpoints, custom success/failure thresholds, and automatic status updates.
* **Granular Route Timeouts**: Powered by `chi`'s contextual deadlines to enforce strict SLAs per route and avoid hanging connections.
* **IP-based Token-Bucket Rate Limiting**: Built on top of `golang.org/x/time/rate` with configurable request bursts and fill rates per route.
* **Robust Graceful Shutdown**: Listens for OS interrupt signals (`SIGINT`/`SIGTERM`), stops background ticks (for rate limits & health checks), and drains active HTTP connections with a 10-second grace window.
* **Path Rewriting**: Supports optional prefix stripping (`strip_prefix: true`), allowing neat modular routes (e.g. mapping gateway route `/api/users/profile` to backend server route `/profile`).

---

## 🛠️ Configuration & Strict Startup Validation

Nexus is designed to fail fast. Rather than starting with an invalid configuration and throwing panic errors at runtime, the gateway parses and strictly validates your rules at boot time.

### Configuration Schema (`configs/config.yml`)

```yaml
port: 8080

health_check:
  path: /health
  interval: 5s
  timeout: 2s
  healthy_threshold: 2
  unhealthy_threshold: 3
  expected_status: 200

routes:
  - path: /api/users
    methods: [GET, POST]
    strip_prefix: true
    timeout: 10s
    backends:
      - url: http://localhost:8081
    rate_limit:
      requests: 100
      per: 1m

  - path: /api/orders
    methods: [GET]
    strip_prefix: true
    timeout: 5s
    backends:
      - url: http://localhost:8082
    rate_limit:
      requests: 50
      per: 1m
```

### Strict Validation Matrix

| Field | Rule | Error Outcome at Startup |
| :--- | :--- | :--- |
| `port` | Must be an integer between `1` and `65535` | Gateway refuses to start |
| `routes` | Must contain at least one route item | Gateway refuses to start |
| `routes[].path` | Must be non-empty and start with a `/` prefix | Gateway refuses to start |
| Duplicate Paths | Identical route paths are rejected | Gateway refuses to start |
| `methods` | Must be a non-empty list of: `GET`, `POST`, `PUT`, `DELETE`, `PATCH` | Gateway refuses to start |
| `backends` | Must list at least one upstream backend URL | Gateway refuses to start |
| `backends[].url`| Must parse as a valid URI containing a schema (`http`/`https`) and host | Gateway refuses to start |
| `rate_limit.per`| Must parse as a positive Go duration string (e.g. `1m`, `30s`) | Gateway refuses to start |
| `timeout` | Must parse as a positive Go duration string (e.g. `5s`) | Gateway refuses to start |
| `health_check` | Top-level active check is optional, but validation rules apply if defined | Gateway refuses to start |
| `health_check.interval` | Must be greater than `0` and larger than `health_check.timeout` | Gateway refuses to start |
| `health_check.expected_status`| Must represent a valid HTTP status code between `100` and `599` | Gateway refuses to start |

---

## 📝 Observability & Structured Logging (`slog`)

Nexus logs structured data using Go's official `log/slog` library. This allows integration into logging stacks like Grafana Loki, ELK, or Datadog without custom regex parsing.

### Example Log Outputs

#### 🚀 Server Startup & Route Registration
```json
{"time":"2026-07-08T10:00:00.000Z","level":"INFO","msg":"health checker started","backends":2,"path":"/health","interval":"5s","timeout":"2s"}
{"time":"2026-07-08T10:00:00.001Z","level":"INFO","msg":"route registered","path":"/api/users","methods":["GET","POST"],"backends":1,"rate_limit":true,"timeout":true}
{"time":"2026-07-08T10:00:00.002Z","level":"INFO","msg":"route registered","path":"/api/orders","methods":["GET"],"backends":1,"rate_limit":true,"timeout":true}
{"time":"2026-07-08T10:00:00.002Z","level":"INFO","msg":"server listening","addr":":8080"}
```

#### 🩺 Health Checks & Transitions
```json
{"time":"2026-07-08T10:05:30.000Z","level":"WARN","msg":"backend became unhealthy","backend":"http://localhost:8081","err":"unhealthy status: 500"}
{"time":"2026-07-08T10:05:45.000Z","level":"INFO","msg":"backend recovered","backend":"http://localhost:8081"}
```

#### 🛡️ Rate Limit Rejection
```json
{"time":"2026-07-08T10:06:12.450Z","level":"WARN","msg":"rate limit exceeded","remote_ip":"127.0.0.1","method":"GET","path":"/api/users"}
```

#### 💸 Request Failure in Reverse Proxy
```json
{"time":"2026-07-08T10:07:05.112Z","level":"ERROR","msg":"upstream request failed","backend":"http://localhost:8082","method":"GET","path":"/api/orders","err":"context deadline exceeded"}
```

#### 📥 Normal Request Completion
```json
{"time":"2026-07-08T10:07:05.115Z","level":"INFO","msg":"request completed","method":"GET","path":"/api/orders","remote_addr":"127.0.0.1:51234","user_agent":"curl/7.81.0","status":200,"bytes_written":42,"duration_ms":3.14}
```

---

## 🚨 Error Semantics & Troubleshooting

When a client receives an HTTP error status code from Nexus, the underlying cause matches standard gateway architecture conventions:

* **`429 Too Many Requests`**:
  * **Cause**: The client IP address exceeded the rate limits configured for that route.
  * **Troubleshooting**: Reduce request frequency, distribute load, or adjust the limits (`requests` and `per`) in `configs/config.yml`.
* **`502 Bad Gateway`**:
  * **Cause**: The selected backend service is offline, refused the connection, or returned an unreadable response.
  * **Troubleshooting**: Verify that the destination backend service is up and listening on the configured address.
* **`503 Service Unavailable`**:
  * **Cause**: All configured backends for a route are marked unhealthy by the background health checker.
  * **Troubleshooting**: Check target backend servers' health and `/health` response codes. Ensure the endpoints return the status code matching `expected_status` (default `200`).
* **`504 Gateway Timeout`**:
  * **Cause**: The backend was reachable but failed to return a response within the configured timeout limit, or the client context timed out.
  * **Troubleshooting**: Optimize backend database/API latency or increase the route `timeout` duration in `configs/config.yml`.

---

## 🚀 Getting Started & Local Development

### Prerequisites

* **Go**: Version `1.25.5` or later.
* **make** (optional but recommended for running convenient targets).

### Running the Gateway & Mock Services

We provide a convenient orchestrator inside the `Makefile` to spin up a full multi-tier microservice simulation locally (Gateway + User service mock + Order service mock):

#### 1. Build the gateway and mock servers:
```bash
make build
```
This writes standalone binary executables into the `bin/` directory:
* `bin/gateway`
* `bin/users-service`
* `bin/orders-service`

#### 2. Run the full environment concurrently:
```bash
make run
```
Or, to run the pre-built binaries:
```bash
make run-built
```

#### 3. Test routing through the gateway:
In another terminal, test routing directly through the Gateway (Port `8080`):
```bash
# Check global Gateway health
curl -i http://localhost:8080/health

# Access Users service (Port 8081) routed and stripped through Gateway
curl -i http://localhost:8080/api/users/

# Access Orders service (Port 8082) routed and stripped through Gateway
curl -i http://localhost:8080/api/orders/
```

---

### Running Tests & Race Detection

The test suite contains strict table-driven unit tests and high-concurrency race tests simulating 10,000 concurrent requests over the reverse proxy.

Run the unit and concurrency test suite with the race detector enabled:
```bash
go test -race -v ./...
```

---

## 🧩 Third-Party Dependencies Philosophy

Nexus adheres strictly to Go's core philosophy of keeping dependencies sparse and light to avoid supply chain vulnerabilities and bloated compiled binaries.

| Dependency | Purpose | Why Not Standard Library? |
| :--- | :--- | :--- |
| `github.com/go-chi/chi/v5` | Router & Middleware Engine | Provides standard-library compatible `http.Handler` routing with sub-routing capabilities and path variable pattern matching. Zero allocations during routing. |
| `gopkg.in/yaml.v3` | YAML Unmarshaler | Go's standard library does not support YAML serialization out of the box. `yaml.v3` is the de facto standard in the Go ecosystem. |
| `golang.org/x/time` | Token-Bucket Limiter | Highly optimized, production-tested atomic token-bucket implementation maintained by the Go core team. |

---

## 🗺️ Future Roadmap

While Nexus is already a production-capable gateway, we have planned a roadmap to expand its capabilities into high-volume distributed environments:

1. **Prometheus Instrumentation**: Expose a `/metrics` route containing histograms for route latencies, request counters labeled by status and route, and active connection gauges.
2. **Circuit Breaking (per backend)**: Implement an automated state-machine circuit breaker (Closed, Open, Half-Open) to fail-fast and avoid overloading struggling upstream services.
3. **JWT Authentication**: Add a verification middleware that extracts Bearer Tokens, validates claims, and injects user identity metadata into the request context.
4. **Containerization & Orchestration**: Include a multi-stage `Dockerfile` (yielding minimal image sizes < 15MB) and `docker-compose.yml` configurations for instant local deployment.
