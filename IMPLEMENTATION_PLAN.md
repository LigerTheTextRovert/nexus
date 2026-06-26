# Nexus — Implementation Plan & Roadmap

**Goal:** Level up Nexus from a clean reverse-proxy into a production-aware API gateway that makes the author hireable as a **junior Go backend developer**, while laying groundwork toward **distributed systems** work.

**Target window:** 1–2 weeks. Build each item as a self-contained PR so the git history tells the story.

---

## Status Snapshot (already built)

These are **done** — don't re-plan them:

- chi router + `httputil.ReverseProxy` with path rewriting (`strip_prefix`)
- YAML-driven routing config + validation (port range, URL scheme/host, paths, HTTP methods, duplicate detection)
- Thread-safe **round-robin load balancer** (atomic counter, multi-backend per route)
- Structured JSON logging middleware (`slog`: method, path, status, duration, IP, UA)
- Proxy error handling (504 on timeout, 502 on upstream failure) + tuned `http.Transport`
- Graceful shutdown (SIGINT/SIGTERM, 10s drain)
- Table-driven tests + a 10k-goroutine concurrency test
- Mock backend services (users/orders) + Makefile

---

## Phase 0 — Fix Existing Issues (½ day, do first)

An interviewer will read this code. These are small but undercut an otherwise clean project.

### 0.1 Enforce HTTP methods (currently dead config)

- **Problem:** `methods` is parsed in `loader.go` and validated in `validator.go`, but `server.go` registers `r.Handle("/*", handler)` — which serves _all_ methods. The config field does nothing.
- **Fix:** Register per-method (`r.Method(m, "/*", handler)`) so the config is enforced, or remove the field. Dead config is a review red flag.

### 0.2 Stop swallowing errors in `routes()`

- **Problem:** `server.routes()` returns `nil` when `NewLoadBalancerHandler` fails, so the server can start with no routes and no error.
- **Fix:** Propagate the error — `New()` / `routes()` should return `error`, and `main` should fail loudly.

### 0.3 Cleanups

- Remove the duplicate `len(c.Routes) == 0` check in `Validate()`.
- `go mod tidy` to drop the `// indirect` tags on direct deps.
- In `load_balancer.go`, `fmt.Errorf("...%q", b)` formats a struct — use `b.Url`.

---

## Phase 1 — Production-Aware Signals (Tier 1, ~3.5 days)

The highest resume-value-per-hour work. Do all four.

### 1.1 Prometheus Metrics + `/metrics` endpoint — **highest value**

- **Build:** request count, latency histogram, error rate, in-flight gauge (labels: route, method, status). Expose `/metrics` via `prometheus/client_golang`.
- **Why:** Observability is table-stakes in 2026. "I instrumented it and can show p99 latency" beats any single feature.
- **Effort:** 1 day

### 1.2 Rate Limiting Middleware (token bucket, per-IP / per-route)

- **Build:** token-bucket limiter (`golang.org/x/time/rate` or hand-rolled with `sync.Mutex`), keyed by client IP or route; return `429` + `Retry-After`. Limits configurable in YAML.
- **Why:** Demonstrates concurrency control over shared state — the #1 thing Go interviews probe.
- **Effort:** 1 day

### 1.3 JWT Authentication Middleware

- **Build:** extract bearer token, validate signature + `exp`/`nbf`, inject claims into `request.Context`. Per-route opt-in via config.
- **Why:** Middleware design, security awareness, `context` usage — all common real-world tasks.
- **Effort:** 1 day

### 1.4 Dockerize (multi-stage) + docker-compose

- **Build:** multi-stage `Dockerfile` (small final image); `docker-compose.yml` running gateway + both mock backends. "Clone and `docker compose up`."
- **Why:** Makes a reviewer actually run it. Multi-stage build signals ops literacy.
- **Effort:** ½ day

---

## Phase 2 — Distributed-Systems Flavor (Tier 2, ~3 days)

These point directly at the longer-term distributed-systems goal.

### 2.1 Active Health Checks + Dead-Backend Ejection

- **Build:** background goroutine probes each backend (`GET /health` or TCP); LB skips unhealthy backends and re-adds them on recovery. Concurrency-safe backend state.
- **Why:** Turns round-robin into a _real_ load balancer.
- **Effort:** 1–1.5 days

### 2.2 Circuit Breaker (per backend)

- **Build:** closed → open → half-open state machine (`sony/gobreaker` or hand-rolled). Trip on failure-rate threshold, fail fast while open, probe in half-open. Pairs with 2.1.
- **Why:** The marquee resilience pattern; core distributed-systems vocabulary.
- **Effort:** 1 day

### 2.3 Request ID Middleware

- **Build:** generate/propagate `X-Request-ID`, add it to every log line.
- **Why:** Cheap; shows understanding of correlation/tracing.
- **Effort:** ~2 hours

---

## Phase 3 — Flagship Polish (Tier 3, ~1 day)

### 3.1 GitHub Actions CI

- Build + `go test -race ./...` + `golangci-lint`. Add a status badge to the README.
- **Why:** A green CI badge signals disciplined shipping.

### 3.2 Load-Test Report

- Run `hey` / `vegeta` against the gateway; put a throughput + p50/p99 table (ideally a graph) in the README.
- **Why:** Concrete numbers ("~12k req/s, p99 4 ms") are what reviewers remember.

---

## Recommended Scope for 1–2 Weeks

Don't build all of it. The sweet spot:

> **Phase 0 (all) + Phase 1 (all) + 2.1 + 2.2 + 3.2**

That covers concurrency, resilience, observability, security, and ops — the junior-backend checklist — with the distributed-systems patterns that point at the next step.

### Target resume bullet

> **Nexus — Go API Gateway**
> Built a concurrent HTTP API gateway in Go: round-robin load balancing with active health checks and circuit breaking, token-bucket rate limiting, JWT auth, and Prometheus-instrumented observability. Containerized with multi-stage Docker; CI with race detection and linting. Sustained ~X k req/s at p99 < Y ms (verified with `vegeta`).

---

## Engineering Principles (apply throughout)

- **Idiomatic Go** — favor the standard library; clear names; wrap errors with `%w`.
- **Concurrency safety** — synchronize all shared state; use `context.Context` for timeouts/cancellation; keep the `-race` test suite green.
- **Testing** — table-driven tests for each new middleware; an integration test against a dummy backend.
- **Structure** — keep the `cmd/` `internal/` `pkg/` layout; each feature is its own package + PR.
- **Docs** — update `README.md` as features land; comment non-obvious logic.
