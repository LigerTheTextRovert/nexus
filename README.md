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
       +-- GET /health  --> {"status": "healthy"}
       +-- GET /        --> "Gateway is running..."
       +-- /api/users/* --> [reverse proxy] --> http://localhost:8081 (strip_prefix)
       +-- /api/orders/* --> [reverse proxy] --> http://localhost:8082 (strip_prefix)
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

- **YAML-based route configuration** — define backends and path prefixes declaratively
- **Reverse proxy** — forwards requests to upstream services using `net/http/httputil.ReverseProxy`
- **Path rewriting** — optionally strips matched path prefixes before forwarding
- **Structured JSON logging** — every request logs method, path, status, duration, user agent, and remote IP
- **Config validation** — validates port range, backend URL schemes/hosts, and path formats at startup; deduplicates routes
- **Graceful shutdown** — handles `SIGINT`/`SIGTERM` with a 10-second drain window
- **Table-driven tests** — for proxy behavior and config validation
- **Concurrency safety test** — 10,000 concurrent requests through the proxy handler

---

## Configuration

`configs/config.yml`:

```yaml
port: 8080
routes:
  - path: /api/users
    backend_URL: http://localhost:8081
    strip_prefix: true
  - path: /api/orders
    backend_URL: http://localhost:8082
    strip_prefix: true
```

Each `Route` maps a URL path prefix to a backend service. When `strip_prefix: true`, the matched prefix is removed before forwarding. For example, a request to `/api/users/42` becomes `/users/42` at the backend.

### Validation Rules

| Field | Rule |
|---|---|
| `port` | Must be between 1 and 65535 |
| `routes` | At least one route required |
| `path` | Must be non-empty and start with `/` |
| `backend_URL` | Must be a valid URI with `http` or `https` scheme and a non-empty host |
| Duplicate paths | Logged as warning and deduplicated (first occurrence wins) |

---

## How It Works

### Startup Flow

1. **Load** — `config.LoadConfig` reads the YAML file and unmarshals it into a `Config` struct
2. **Validate** — `cfg.Validate()` checks port range, route presence, backend URLs, and paths
3. **Build** — `server.New(&cfg)` creates the chi router, attaches logging middleware, registers `/health` and `/`, then dynamically registers a `chi.Route` for each configured backend
4. **Serve** — `server.Start()` launches `ListenAndServe`, blocks on OS signals, then calls `shutdown()` with a 10-second grace period

### Request Lifecycle

1. Request arrives at the chi router
2. `LoggingMiddleware` wraps the response writer to capture status and bytes
3. Chi matches the route and dispatches to the corresponding `ProxyHandler`
4. The handler optionally rewrites the path (strip prefix), then calls `httputil.ReverseProxy.ServeHTTP`
5. The response flows back through the middleware chain
6. The logging middleware marshals the request details to JSON and logs them

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

All other functionality uses the Go standard library (`net/http`, `net/http/httputil`, `context`, `encoding/json`, `os/signal`).
