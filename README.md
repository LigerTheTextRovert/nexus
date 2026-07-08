<p align="center">
  <img src="assets/logo.png" alt="Nexus Logo" width="180">
</p>

<h1 align="center">Nexus</h1>

<p align="center">
    A lightweight HTTP API gateway / reverse proxy written in Go. Nexus sits in front of backend services and routes incoming HTTP requests to upstreams based on a YAML configuration file.
</p>

> /ˈneksəs/ — A connection or series of connections linking two or more things

---

## Architecture

```
Client Request
       |
       v
  [chi router] -- JSON structured logging
       |
       +-- GET /health  --> {"status": "healthy"}
       +-- GET /         --> "Gateway is running..."
       +-- /api/users/*  -- [timeout: 10s] -- [rate limit: 100 req/min per IP]
       |                     --> [health-aware round-robin reverse proxy]
       |                         --> healthy backend: http://localhost:8081 (strip_prefix)
       +-- /api/orders/* -- [timeout: 5s] -- [rate limit: 50 req/min per IP]
                             --> [health-aware round-robin reverse proxy]
                                 --> healthy backend: http://localhost:8082 (strip_prefix)

Background:
  [health checker] -- every 5s --> GET {backend}/health
       |
       +-- expected 200 after 2 successes  --> mark backend healthy
       +-- 3 consecutive failures          --> mark backend unhealthy
```

### Project Layout

```
nexus/
  cmd/
    gateway/main.go             # Gateway entrypoint: loads config, validates it, starts the server
    users-service/main.go       # Demo upstream service on :8081
    orders-service/main.go      # Demo upstream service on :8082
  configs/config.yml            # Runtime configuration
  internal/
    config/
      loader.go                 # YAML config structs and loading
      validator.go              # Config validation for ports, routes, methods, URLs, rate limits
      validator_test.go         # Table-driven config validation tests
    health/
      checker.go                # Active backend health checker with success/failure thresholds
    logging/
      middleware.go             # JSON request logging middleware
    proxy/
      proxy.go                  # Reverse proxy setup, transport, and upstream error handling
      load_balancer.go          # Thread-safe health-aware round-robin backend selection and path rewriting
      proxy_test.go             # Proxy behavior tests
      proxy_concurrent_test.go  # 10k goroutine concurrency test
      load_balancer_test.go     # Load balancer tests
    rate-limit/
      middleware.go             # Per-IP token bucket rate limiter with background cleanup
      middleware_test.go        # Rate limiter tests
    server/
      server.go                 # HTTP server, route wiring, graceful shutdown
      handler.go                # Health/root handlers
  pkg/utils/                    # Public utility helpers
  Makefile                      # Convenience run/build targets
```

---

## Features

- **YAML-based route configuration** — define route prefixes, allowed HTTP methods, upstream backends, optional timeouts, optional rate limits, and path rewriting declaratively
- **Health-aware round-robin load balancing** — routes can define one or more backends; requests are distributed with an atomic counter and unhealthy backends are skipped
- **Active backend health checks** — a background checker periodically probes each backend health endpoint, tracks consecutive successes/failures, and marks backends healthy or unhealthy using atomic state
- **Reverse proxying** — forwards requests with `net/http/httputil.ReverseProxy` and a tuned `http.Transport`
- **Path rewriting** — optionally strips the matched route prefix before forwarding; for example `/api/users/42` can become `/42` upstream
- **Per-route rate limiting** — optional token bucket limiter per client IP, configured independently per route (`requests` and `per` window), with idle clients cleaned up in the background
- **Per-route timeouts** — optional `timeout` values use chi middleware to bound request handling for a route
- **Structured JSON logging** — gateway startup, route registration, health-check transitions, request completion, rate-limit rejections, and upstream errors are logged with `log/slog`
- **Startup config validation** — validates port range, route presence, duplicate paths, HTTP methods, backend URLs, rate limit values, timeouts, and health-check settings before serving traffic
- **Graceful shutdown** — handles `SIGINT`/`SIGTERM`, cancels rate limiter cleanup and health-check goroutines, and drains the server with a 10-second grace period
- **Tests** — table-driven validation/proxy tests plus concurrency coverage for the proxy/load balancer path

---

## Configuration

`configs/config.yml`:

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

Each route maps a URL path prefix to one or more backend services. When `strip_prefix: true`, the matched prefix is removed before forwarding. For example, a request to `/api/users/42` becomes `/42` at the backend.

`methods` controls which HTTP verbs are accepted on that route. `rate_limit` and `timeout` are optional. When `rate_limit` is omitted, the route is not rate limited. When multiple `backends` are configured, Nexus chooses the next **healthy** backend with round-robin selection.

The top-level `health_check` section enables active backend probing. Nexus calls `GET {backend.url}{health_check.path}` on every configured backend at the configured interval. Backends start optimistically healthy, become unhealthy after `unhealthy_threshold` consecutive failures, and recover after `healthy_threshold` consecutive successful checks with `expected_status`.

### Validation Rules

| Field                              | Rule                                                                                      |
| ---------------------------------- | ----------------------------------------------------------------------------------------- |
| `port`                             | Must be between 1 and 65535                                                               |
| `routes`                           | At least one route is required                                                            |
| `path`                             | Must be non-empty and start with `/`                                                      |
| Duplicate route paths              | Rejected at startup                                                                       |
| `methods`                          | At least one method is required; valid values are `GET`, `POST`, `PUT`, `DELETE`, `PATCH` |
| `backends`                         | At least one backend is required                                                          |
| `backends[].url`                   | Must be a valid URI with `http` or `https` scheme and a non-empty host                    |
| `rate_limit.requests`              | Optional; when present, must be > 0                                                       |
| `rate_limit.per`                   | Optional; when present, must parse as a positive Go duration such as `1m` or `30s`        |
| `timeout`                          | Optional; when present, must parse as a positive Go duration                              |
| `health_check.path`                | Optional top-level section; when present, path must be non-empty and start with `/`       |
| `health_check.interval`            | Must be greater than `0` and greater than `health_check.timeout`                          |
| `health_check.timeout`             | Must be greater than `0`                                                                  |
| `health_check.healthy_threshold`   | Must be at least `1`                                                                      |
| `health_check.unhealthy_threshold` | Must be at least `1`                                                                      |
| `health_check.expected_status`     | Must be a valid HTTP status code (`100`-`599`)                                            |

---

## How It Works

### Startup Flow

1. **Load** — `config.LoadConfig` reads `configs/config.yml` and unmarshals it into `config.Config`.
2. **Validate** — `cfg.Validate()` checks port range, route presence, duplicate paths, methods, backend URLs, optional rate limit values, optional route timeouts, and optional health-check settings.
3. **Build** — `server.New(&cfg)` creates a lifecycle context, registers global logging middleware, health/root handlers, and each configured route.
4. **Wire route middleware** — each route gets its own health-aware load balancer, optional timeout middleware, and optional rate limiter manager.
5. **Start health checks** — if `health_check` is configured, a background checker starts probing all configured backends and updating their atomic health state.
6. **Serve** — `server.Start()` launches `ListenAndServe`, waits for `SIGINT` or `SIGTERM`, then shuts down with a 10-second grace period.

### Request Lifecycle

1. Request arrives at the chi router.
2. `LoggingMiddleware` wraps the response writer to capture status and bytes.
3. Chi matches the configured route and method.
4. If configured, the route timeout middleware applies a request deadline.
5. If configured, the route's `RateLimiterManager.Middleware` checks the client IP's token bucket.
   - If the bucket is empty, Nexus logs a warning and returns `429 Too Many Requests`.
6. The request reaches the route's `LoadBalancer`.
7. The load balancer atomically selects the next healthy backend, skipping backends marked unhealthy by the health checker.
8. If no healthy backend is available, Nexus logs an error and returns `503 Service Unavailable`.
9. If a healthy backend exists, the load balancer optionally strips the route prefix and delegates to that backend's reverse proxy.
10. If the upstream fails, the reverse proxy error handler returns `504 Gateway Timeout` for deadline errors or `502 Bad Gateway` for other upstream failures.
11. The response flows back through the middleware chain and the logging middleware records request metadata.

---

## Getting Started

### Prerequisites

- Go 1.25.5 or compatible with the version in `go.mod`

### Run the gateway only

```bash
go run ./cmd/gateway
```

The gateway starts on port `8080` by default, as configured in `configs/config.yml`. This assumes the configured upstream services are already running.

### Run demo upstreams and gateway

```bash
make run
```

This starts:

- `users-service` on `:8081`
- `orders-service` on `:8082`
- `gateway` on `:8080`

Example requests:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/users/
curl http://localhost:8080/api/orders/
```

The demo upstreams also expose `/health` on `:8081` and `:8082`, which the gateway's active health checker probes in the background.

### Build

```bash
make build
```

Binaries are written to `bin/`:

- `bin/gateway`
- `bin/users-service`
- `bin/orders-service`

Run the built binaries together with:

```bash
make run-built
```

### Test

```bash
go test -race -v ./...
```

---

## Used Libraries

| Library                                                             | Purpose                                         |
| ------------------------------------------------------------------- | ----------------------------------------------- |
| [go-chi/chi/v5](https://github.com/go-chi/chi)                      | Lightweight HTTP router with middleware support |
| [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)             | YAML config file parsing                        |
| [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) | Token bucket rate limiter                       |

Most other functionality uses the Go standard library (`net/http`, `net/http/httputil`, `context`, `log/slog`, `os/signal`, `sync`, `sync/atomic`).
