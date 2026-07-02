# Nexus

> /ˈneksəs/ — A connection or series of connections linking two or more things

A lightweight HTTP API gateway / reverse proxy written in Go. Nexus sits in front of backend services and routes incoming HTTP requests to upstreams based on a YAML configuration file.

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
       |                     --> [round-robin reverse proxy] --> http://localhost:8081 (strip_prefix)
       +-- /api/orders/* -- [timeout: 5s] -- [rate limit: 50 req/min per IP]
                             --> [round-robin reverse proxy] --> http://localhost:8082 (strip_prefix)
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
    logging/
      middleware.go             # JSON request logging middleware
    proxy/
      proxy.go                  # Reverse proxy setup, transport, and upstream error handling
      load_balancer.go          # Thread-safe round-robin backend selection and path rewriting
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
- **Round-robin load balancing** — routes can define one or more backends; requests are distributed with an atomic counter for concurrent safety
- **Reverse proxying** — forwards requests with `net/http/httputil.ReverseProxy` and a tuned `http.Transport`
- **Path rewriting** — optionally strips the matched route prefix before forwarding; for example `/api/users/42` can become `/42` upstream
- **Per-route rate limiting** — optional token bucket limiter per client IP, configured independently per route (`requests` and `per` window), with idle clients cleaned up in the background
- **Per-route timeouts** — optional `timeout` values use chi middleware to bound request handling for a route
- **Structured JSON logging** — gateway startup, route registration, request completion, rate-limit rejections, and upstream errors are logged with `log/slog`
- **Startup config validation** — validates port range, route presence, duplicate paths, HTTP methods, backend URLs, and rate limit values before serving traffic
- **Graceful shutdown** — handles `SIGINT`/`SIGTERM`, cancels rate limiter cleanup goroutines, and drains the server with a 10-second grace period
- **Tests** — table-driven validation/proxy tests plus concurrency coverage for the proxy/load balancer path

---

## Configuration

`configs/config.yml`:

```yaml
port: 8080

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

`methods` controls which HTTP verbs are accepted on that route. `rate_limit` and `timeout` are optional. When `rate_limit` is omitted, the route is not rate limited. When multiple `backends` are configured, Nexus chooses the next backend with round-robin selection.

### Validation Rules

| Field | Rule |
|---|---|
| `port` | Must be between 1 and 65535 |
| `routes` | At least one route is required |
| `path` | Must be non-empty and start with `/` |
| Duplicate route paths | Rejected at startup |
| `methods` | At least one method is required; valid values are `GET`, `POST`, `PUT`, `DELETE`, `PATCH` |
| `backends` | At least one backend is required |
| `backends[].url` | Must be a valid URI with `http` or `https` scheme and a non-empty host |
| `rate_limit.requests` | Optional; when present, must be > 0 |
| `rate_limit.per` | Optional; when present, must parse as a positive Go duration such as `1m` or `30s` |

---

## How It Works

### Startup Flow

1. **Load** — `config.LoadConfig` reads `configs/config.yml` and unmarshals it into `config.Config`.
2. **Validate** — `cfg.Validate()` checks port range, route presence, duplicate paths, methods, backend URLs, and optional rate limit values.
3. **Build** — `server.New(&cfg)` creates a lifecycle context, registers global logging middleware, health/root handlers, and each configured route.
4. **Wire route middleware** — each route gets its own load balancer, optional timeout middleware, and optional rate limiter manager.
5. **Serve** — `server.Start()` launches `ListenAndServe`, waits for `SIGINT` or `SIGTERM`, then shuts down with a 10-second grace period.

### Request Lifecycle

1. Request arrives at the chi router.
2. `LoggingMiddleware` wraps the response writer to capture status and bytes.
3. Chi matches the configured route and method.
4. If configured, the route timeout middleware applies a request deadline.
5. If configured, the route's `RateLimiterManager.Middleware` checks the client IP's token bucket.
   - If the bucket is empty, Nexus logs a warning and returns `429 Too Many Requests`.
6. The request reaches the route's `LoadBalancer`.
7. The load balancer atomically selects the next backend, optionally strips the route prefix, and delegates to the backend's reverse proxy.
8. If the upstream fails, the reverse proxy error handler returns `504 Gateway Timeout` for deadline errors or `502 Bad Gateway` for other upstream failures.
9. The response flows back through the middleware chain and the logging middleware records request metadata.

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

| Library | Purpose |
|---|---|
| [go-chi/chi/v5](https://github.com/go-chi/chi) | Lightweight HTTP router with middleware support |
| [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) | YAML config file parsing |
| [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) | Token bucket rate limiter |

Most other functionality uses the Go standard library (`net/http`, `net/http/httputil`, `context`, `log/slog`, `os/signal`, `sync`, `sync/atomic`).
