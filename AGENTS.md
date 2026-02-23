# Repository Guidelines

## Project Structure & Module Organization

- `cmd/gateway/main.go`: application entrypoint for the HTTP gateway.
- `internal/config`: YAML config loading and config structs.
- `internal/proxy`: reverse-proxy handler wiring.
- `internal/logging`: request logging middleware.
- `pkg/utils`: shared utility helpers.
- `configs/config.yml`: local runtime configuration (`port`, `routes`).

Keep gateway-specific wiring in `cmd/` and reusable internals in `internal/` or `pkg/`.

## Build, Test, and Development Commands

- `go run ./cmd/gateway`: run the gateway locally (reads `configs/config.yml`).
- `go build ./cmd/gateway`: compile the gateway binary.
- `go test ./...`: run all tests across modules.
- `go test -cover ./...`: run tests with coverage output.
- `go fmt ./... && go vet ./...`: format and statically check code before PR.

Example local run flow:

```bash
go fmt ./...
go test ./...
go run ./cmd/gateway
```

## Coding Style & Naming Conventions

- Follow standard Go style and formatting (`gofmt` required).
- Use tabs/`gofmt` defaults for indentation; do not hand-align spacing.
- Package names: short, lowercase, no underscores (e.g., `proxy`, `config`).
- Exported identifiers use `PascalCase`; unexported identifiers use `camelCase`.
- Keep handlers/middleware small and composable; route setup belongs in `main.go`.

## Testing Guidelines

- Place tests next to implementation files as `*_test.go`.
- Prefer table-driven tests for handlers, config loading, and utilities.
- Cover critical paths: YAML parsing, route registration behavior, and proxy handler responses.
- Run `go test ./...` locally before opening a PR.

## Commit & Pull Request Guidelines

- Follow Conventional Commit-style prefixes already used in history:
  - `fix: ...`, `refactor: ...`, `chore: ...`
- Keep commit messages imperative and focused on one change.
- PRs should include:
  - a short problem/solution summary,
  - linked issue (if available),
  - test evidence (`go test ./...` output),
  - config or behavior notes for route changes.

## Configuration Tips

- Keep `configs/config.yml` environment-safe; do not commit secrets.
- Validate `backend_url` and `path` values before deploying config changes.
