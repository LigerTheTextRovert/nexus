# Nexus

> /ˈneksəs/ — A connection or series of connections linking two or more things

A lightweight HTTP API gateway / reverse proxy written in Go. Nexus sits in front of multiple backend services and routes incoming HTTP requests to the correct upstream based on a YAML configuration file.

---

## Architecture

```
Client Request
       |
       v
  [chi router]  -- logging middleware (JSON structured logs)
       |
       +-- GET /health   --> {"status": "healthy"}
       +-- GET /          --> "Gateway is running..."
       +-- /api/users/*  -- [rate limit: 100 req/min per IP]
       |                      --> [reverse proxy] --> http://localhost:8081 (strip_prefix)
       +-- /api/orders/* -- [rate limit: 50 req/min per IP]
                             --> [reverse proxy] --> http://localhost:8082 (strip_prefix)
```

### Project Layout

```
nexus/
  cmd/gateway/main.go              # Entrypoint - wires config to server
  internal/
    config/
      loader.go                     # YAML config loading
      validator.go                  # Config validation (port, routes, URLs)
      validator_test.go             # Table-driven tests
    proxy/
      proxy.go                      # Reverse proxy handler with path rewriting
      proxy_test.go                 # Table-driven proxy tests
      proxy_concurrent_test.go      # 10k goroutine concurrency test
    rate-limit/
      middleware.go                 # Per-IP token bucket rate limiter with background cleanup
    server/
      server.go                     # HTTP server, route wiring, graceful shutdown
      handler.go                    # Placeholder for future handler logic
    logging/
      middleware.go                 # HTTP logging middleware (JSON structured logs)
      logger.go                     # Placeholder for future logger extensions
  pkg/utils/                        # Placeholder for public utility functions
  configs/config.yml                # Runtime configuration
```

---

## Features

- **YAML-based route configuration** — define backends, allowed HTTP methods, timeouts, and path prefixes declaratively
- **Per-route rate limiting** — token bucket limiter per client IP, configured independently per route (`requests` and `per` window); idle clients cleaned up in the background
- **Reverse proxy** — forwards requests to upstream services using `net/http/httputil.ReverseProxy`
- **Path rewriting** — optionally strips matched path prefixes before forwarding
- **Structured JSON logging** — every request logs method, path, status, duration, user agent, and remote IP; rate-limited requests emit a `WARN` log before returning `429`
- **Config validation** — validates port range, backend URL schemes/hosts, and path formats at startup; deduplicates routes
- **Graceful shutdown** — handles `SIGINT`/`SIGTERM`, cancels rate limiter cleanup goroutines, then drains with a 10-second grace period
- **Table-driven tests** — for proxy behavior and config validation
- **Concurrency safety test** — 10,000 concurrent requests through the proxy handler

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

Each `Route` maps a URL path prefix to one or more backend services. When `strip_prefix: true`, the matched prefix is removed before forwarding. For example, a request to `/api/users/42` becomes `/42` at the backend. `methods` restricts which HTTP verbs are accepted on that route. `rate_limit` defines an independent token bucket per client IP for that route.

### Validation Rules

| Field | Rule |
|---|---|
| `port` | Must be between 1 and 65535 |
| `routes` | At least one route required |
| `path` | Must be non-empty and start with `/` |
| `backends[].url` | Must be a valid URI with `http` or `https` scheme and a non-empty host |
| `methods` | Must be valid HTTP verbs (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`) |
| `rate_limit.requests` | Must be > 0 |
| `rate_limit.per` | Must be > 0 (e.g. `1m`, `30s`) |
| Duplicate paths | Logged as warning and deduplicated (first occurrence wins) |

---

## How It Works

### Startup Flow

1. **Load** — `config.LoadConfig` reads the YAML file and unmarshals it into a `Config` struct
2. **Validate** — `cfg.Validate()` checks port range, route presence, backend URLs, and paths
3. **Build** — `server.New(&cfg)` creates a shared lifecycle context, instantiates one `RateLimiterManager` per route (from `rate_limit` config), registers the chi router with logging middleware, `/health`, `/`, and each proxy route with its corresponding rate limiter
4. **Serve** — `server.Start()` launches `ListenAndServe`, blocks on OS signals, then calls `shutdown()` with a 10-second grace period

### Request Lifecycle

1. Request arrives at the chi router
2. `LoggingMiddleware` wraps the response writer to capture status and bytes
3. Chi matches the route; the per-route `RateLimiterManager.Middleware` checks the client IP's token bucket
   - If the bucket is empty: logs a `WARN` and returns `429 Too Many Requests` immediately
4. The request is dispatched to the corresponding `ProxyHandler`
5. The handler optionally rewrites the path (strip prefix), then calls `httputil.ReverseProxy.ServeHTTP`
6. The response flows back through the middleware chain
7. The logging middleware records method, path, status, duration, remote IP, and user agent

---

## Getting Started

### Prerequisites

- Go 1.22+

### Run

```bash
go run ./cmd/gateway/
```

The server starts on port 8080 (as configured in `configs/config.yml`).

### Test

```bash
go test -race -v ./...
```

---

## Used Libraries

| Library | Purpose |
|---|---|
| [go-chi/chi](https://github.com/go-chi/chi) | Lightweight HTTP router with middleware support |
| [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) | YAML config file parsing |
| [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) | Token bucket rate limiter |

All other functionality uses the Go standard library (`net/http`, `net/http/httputil`, `context`, `log/slog`, `os/signal`, `sync`).
