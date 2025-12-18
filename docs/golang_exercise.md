# Programming concurrent graph analysis in Golang

Implement a HTTP microservice that simulates concurrent energy data aggregation in a grid.

The service exposes two endpoints:

- /graph that describes the nodes af the grid and how they are connected.
- /measurements that receives measurements for each node of the grid.

## Graph endpoint

When a graph is received, the service computes the islands, i.e. the
interconnected nodes.

e.g. with the following payload:

```json
{
  "nodes": ["A", "B", "C", "D"],
  "edges": [
    ["A", "B"],
    ["C", "D"]
  ]
}
```

returns:

```json
{
  "islands": [
    ["A", "B"],
    ["C", "D"]
  ]
}
```

## Measurements endpoint

The measurements end point, supports the pushing of measurements for each node,
one measurement per node:

```json
{
  "node": "A",
  "value": 5.3
}
```

after each measurement update, it returns the up-to-date energy per islands, e.g.,
with the previous graph, and assuming no previous measurement, it should return:

```json
{
  "island": ["A", "B"],
  "total": 5.3
}
```

when this other measurement arrives

```json
{
  "node": "B",
  "value": 10.1
}
```

It should return:

```json
[ 
    {
        "island": ["A", "B"],
        "total": 15.4
    },
    {
        "island": ["C", "D"],
        "total": 0
    }
]
```

if topology changes before the next measurement arrive:

```json
{
  "nodes": ["A", "B", "C", "D"],
  "edges": [
    ["A", "C"],
    ["B", "D"]
  ]
}
```

and we get the measurement:

```json
{
  "node": "C",
  "value": 1.1
}
```

It should return:

```json
[ 
    {
        "island": ["A", "C"],
        "total": 6.4
    },
    {
        "island": ["B", "D"],
        "total": 10.1
    }
]
```

## Notes

1. While you are computing the graph due to an updated, you queue measurement
before processing and returning results.
1. A great number of messages might arrive at once, so make sure to handle
   spikes in messages (see client below).
1. When SIGINT is received, the server finishes ongoing requests and shuts down cleanly

## Validation

To validate the microservice, develop a client that sends messages around every
20ms. Test the microservice with a graph with at least 100 nodes.
