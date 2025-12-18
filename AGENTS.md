# Repository Guidelines

## Project Structure & Module Organization

- Go module `zgrid`; Go 1.25+. Assignment brief lives in `docs/golang_exercise.md`—keep code aligned with that contract. **NEVER EVER modify `docs/golang_exercise.md`.**
- `cmd/server` runs the HTTP microservice (`/graph`, `/measurements`) and wires the event loop into the router. `cmd/client` should drive load (≈20ms between messages) for validation.
- `business` holds grid state (`GridState`), island computation, measurement aggregation, and the event types flowing over channels.
- `api` exposes HTTP handlers plus middleware that injects the shared event channel into request context.
- `foundation` holds HTTP helpers (`Decode`, `Respond`) and middleware scaffolding. `bin/` is the build output; `docs/` contains reference material.
- Place new package-level tests next to code (`*_test.go` in the same package).

## Service Behavior Requirements

- `/graph` accepts nodes + edge pairs, computes islands, and replies with `{"islands":[... ]}`.
- `/measurements` accepts `{ "node": "...", "value": <float> }` and returns per-island totals based on the latest topology.
- Queue measurements while recomputing graphs; handle spikes (expect bursts from the client). Ensure SIGINT triggers graceful shutdown (finish in-flight requests, stop event loop cleanly).

## Build, Test, and Development Commands

- `make` or `make all`: build all binaries to `bin/` with version injected from `git describe`.
- `go run ./cmd/server -addr :8000`: run the HTTP server locally.
- `go run ./cmd/client`: exercise the service; configure it to push messages ~every 20ms and test with ≥100 nodes.
- `make test`: execute all tests (add them as you contribute).
- `make lint`: format and lint before raising a PR.

## Coding Style & Naming Conventions

- Use `gofmt` output (tabs, standard Go casing). Keep handlers small with early validation for method and `Content-Type`.
- Exported types/constants use PascalCase; unexported helpers stay package-local and start lowercase.
- Decode requests via `foundation.Decode` (keeps `DisallowUnknownFields`) and respond with `foundation.Respond`.
- When extending the event loop, keep work quick and non-blocking; buffer reply channels when responses are awaited.

## Testing Guidelines

- Use table-driven tests with the standard `testing` package; cover island computation and measurement aggregation under topology changes.
- Test handlers with `httptest` and stubbed event channels; include cases for bursts and concurrent updates.
- Run `go test -cover ./...` locally; add focused benchmarks (`BenchmarkXxx`) for hot paths if optimized.

## Commit & Pull Request Guidelines

- Follow the existing concise, imperative commit style (e.g., “measurements endpoint”, “server ready”); one logical change per commit.
- PRs should state the problem, the approach, endpoints affected, and tests/commands run (`make`, `go test ./...`). Include curl examples or JSON payloads for new handlers.
- Link issues if applicable; request screenshots or sample responses when changing API surface or behavior.

## Security & Configuration Tips

- The server defaults to `:8000`; keep it behind your local firewall or reverse proxy as there is no TLS/auth. Avoid logging sensitive payloads.
- Validate external input strictly; keep `foundation.Decode`’s `DisallowUnknownFields` behavior when adding schemas.
