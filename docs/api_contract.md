# API Contract Clarification

The assignment brief (`docs/golang_exercise.md`) describes the behavior of the service and provides example payloads. Some examples show the `/measurements` response as a single JSON object, while later examples show a JSON array of per-island totals.

On success (`200 OK`), this implementation **always returns a JSON list (array) from `/measurements`**, including the single-island case. This was a deliberate choice to provide a stable and consistent response shape in the absence of an explicit requirement.

## Endpoints

### `POST /graph`

Request body:

```json
{
  "nodes": ["A", "B", "C", "D"],
  "edges": [
    ["A", "B"],
    ["C", "D"]
  ]
}
```

Response body:

```json
{
  "islands": [
    ["A", "B"],
    ["C", "D"]
  ]
}
```

### `POST /measurements`

Request body:

```json
{
  "node": "A",
  "value": 5.3
}
```

Response body (always a list/array):

```json
[
  { "island": ["A", "B"], "total": 5.3 },
  { "island": ["C", "D"], "total": 0 }
]
```
