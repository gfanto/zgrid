# Golang exercise

HTTP microservice that computes connected components (“islands”) for a grid topology and aggregates per-node measurements into per-island totals.

The functional contract is described in `docs/golang_exercise.md`.

This repository also documents an explicit API response shape choice in `docs/api_contract.md` (without altering the assignment brief).

## Architecture

- `cmd/server`: HTTP server entrypoint (`/graph`, `/measurements`), SIGINT handling, and wiring the shared event channel into the router.
- `cmd/client`: simple load generator that posts a graph once, then posts random measurements on a ticker (~20ms).
- `api`: HTTP handlers and middleware that injects the event channel into the request context.
- `business`: domain model (`Graph`, `Grid`) and the single-threaded event loop that processes updates.
- `foundation`: HTTP helpers (`Decode`, `Respond`) and middleware scaffolding.

## Technical decisions and tradeoffs

### Single-threaded grid state (event loop)

`business.Grid` is mutated only by `Grid.Loop(ctx, events)`, which processes incoming `Event`s serially. This avoids locks and makes state transitions predictable (graph updates and measurement updates can’t interleave).

Tradeoff: throughput depends on how fast each event is processed; long graph recomputations will slow measurement processing. If graph updates become expensive, consider incremental island updates or moving island computation to a separate worker and buffering incoming measurements.

### Graph representation and island computation

- The topology is represented as a node list plus an adjacency list (`map[string][]string`).
- `business.NewGraph` builds an **undirected** adjacency list (adds both `A -> B` and `B -> A`) and ignores malformed edges or edges referencing unknown nodes.
- `computeIslands` uses an iterative DFS to avoid recursion limits.

Tradeoff: island order and node order within islands are deterministic based on input order, but not guaranteed to be sorted.

### Measurement retention across topology changes

Measurements are stored as a per-node “latest value” map and are **retained across graph updates**. Aggregation (`aggregate`) only sums nodes present in the current topology (via `nodeToIsland`), so measurements for absent nodes do not affect totals.

Tradeoff: if node IDs are unbounded, retaining measurements indefinitely can grow memory. A future improvement is to periodically prune measurements for nodes not seen in the last N graphs or to cap map size.

### Backpressure and latency limits

Handlers send events to the grid loop and (optionally) wait for a reply. The shared `events` channel is buffered (`bufferSize` in `cmd/server`) so measurements can queue while island recomputation is in progress, matching the exercise requirement.

Tradeoff: queuing improves correctness under bursts but can increase request latency and memory usage under sustained overload. To avoid unbounded blocking when the queue is full, `/measurements` applies a small enqueue timeout (20ms); if it can’t enqueue the event in time it returns `429 Too Many Requests` with `{ "error": "server busy, try again" }`.

Design choice: this “fail fast when the queue is full” behavior is most likely to show up right after a `/graph` update (while islands are being recomputed) or during a measurement burst. Clients should treat `429` as transient and retry with a small backoff.

### Graceful shutdown

`cmd/server` uses `signal.NotifyContext` and `http.Server.Shutdown` to stop accepting new connections and let in-flight requests finish when SIGINT is received. The grid loop also exits when its context is canceled.

## Build and run

Build binaries to `bin/`:

```bash
make all
```

Run the server (defaults to `:8000`):

```bash
go run ./cmd/server -addr :8000
```

## Using the API

Create/update the topology:

```bash
curl -sS -X POST http://127.0.0.1:8000/graph \
  -H 'Content-Type: application/json' \
  -d '{"nodes":["A","B","C","D"],"edges":[["A","B"],["C","D"]]}'
```

Send a measurement:

```bash
curl -sS -X POST http://127.0.0.1:8000/measurements \
  -H 'Content-Type: application/json' \
  -d '{"node":"A","value":5.3}'
```

Note: on success (`200 OK`), `/measurements` always responds with a JSON list (array) of island totals, even if there is only one island. This is a deliberate choice because the exercise brief does not explicitly mandate whether the single-island case should be an object or a list, and it shows examples of both.

## Testing

Unit tests:

```bash
go test ./...
```

Manual test workflow (what I use locally):

1. Start the server:

```bash
go run ./cmd/server -addr :8000
```

2. In another terminal, start the load client:

```bash
go run ./cmd/client -addr :8000
```

3. While the client is running, “peek” at the current per-island totals with `curl` (and optionally `jq` for pretty output):

```bash
curl -sS -X POST http://127.0.0.1:8000/measurements \
  -H 'Content-Type: application/json' \
  -d '{}' | jq
```

Note: sending `{}` works as a lightweight probe because the empty node is not part of the graph, so it won’t affect totals (as long as you don’t use empty-string node IDs in your topology).

Load test client (posts a random graph once, then measurements every ~20ms):

```bash
go run ./cmd/client -addr :8000
```

`cmd/client` flags let you control graph size and send rate. Defaults are `-nodes 100`, `-edges 150`, `-interval 20ms`:

```bash
go run ./cmd/client -addr :8000 -nodes 200 -edges 400 -interval 20ms
```

## Linting and formatting

```bash
make lint
```
