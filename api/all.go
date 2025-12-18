package api

import (
	"net/http"
	"time"
	"zgrid/business"
	"zgrid/foundation"
)

const backpressureTimeout = 20 * time.Millisecond

// All registers all HTTP routes for the grid service.
func All() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/graph", foundation.WrapMiddleware(http.HandlerFunc(graphHandler),
		foundation.RequireMethod(http.MethodPost),
		foundation.RequireJSONContentType,
	))
	mux.Handle("/measurements", foundation.WrapMiddleware(http.HandlerFunc(measurementsHandler),
		foundation.RequireMethod(http.MethodPost),
		foundation.RequireJSONContentType,
	))

	return mux
}

func graphHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	events := getStateEvents(ctx)
	if events == nil {
		foundation.Respond(w, http.StatusInternalServerError, newErrResp(http.StatusText(http.StatusInternalServerError)))
		return
	}

	// ----------------------------------------------------------------------------
	// Validate Request

	type graphPayload struct {
		Nodes []string `json:"nodes"`
		Edges []Edge   `json:"edges"`
	}
	payload, err := foundation.Decode[graphPayload](w, r)
	if err != nil {
		foundation.Respond(w, http.StatusBadRequest, newErrResp("invalid graph payload"))
		return
	}

	// ---------------------------------------------------------------------------
	// Process Request

	edges := make([][]string, len(payload.Edges))
	for i, edge := range payload.Edges {
		// Edge is a JSON array of exactly 2 node IDs; convert to []string without copying.
		edges[i] = []string(edge)
	}
	graph := business.NewGraph(payload.Nodes, edges)

	resp := make(chan [][]string, 1)
	updateEvent := business.GraphUpdate{
		Graph: graph,
		Reply: resp,
	}

	// ----------------------------------------------------------------------------
	// Send Response

	select {
	case events <- updateEvent:
		// Wait for the recomputation result; measurements queued during this time
		// will be processed once the graph update completes.
		select {
		case islands := <-resp:
			foundation.Respond(w, http.StatusOK, struct {
				Islands [][]string `json:"islands"`
			}{
				Islands: islands,
			})
		case <-ctx.Done():
			foundation.Respond(w, http.StatusRequestTimeout, newErrResp(http.StatusText(http.StatusRequestTimeout)))
			return
		}
	case <-ctx.Done():
		foundation.Respond(w, http.StatusRequestTimeout, newErrResp(http.StatusText(http.StatusRequestTimeout)))
		return
	}
}

func measurementsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	events := getStateEvents(ctx)
	if events == nil {
		foundation.Respond(w, http.StatusInternalServerError, newErrResp(http.StatusText(http.StatusInternalServerError)))
		return
	}

	// ----------------------------------------------------------------------------
	// Validate Request

	type measurementsPayload struct {
		Node  string  `json:"node"`
		Value float64 `json:"value"`
	}
	measurement, err := foundation.Decode[measurementsPayload](w, r)
	if err != nil {
		foundation.Respond(w, http.StatusBadRequest, newErrResp("invalid measurement payload"))
		return
	}

	// ----------------------------------------------------------------------------
	// Process Request

	resp := make(chan []business.IslandMeasurement, 1)
	// If a node not present in the graph is sent anyway to avoid coupling and locking
	updateEvent := business.MeasurementUpdate{
		NodeMeasurement: business.NodeMeasurement{
			Node:  measurement.Node,
			Value: measurement.Value,
		},
		Reply: resp,
	}

	// ----------------------------------------------------------------------------
	// Send Response

	select {
	case events <- updateEvent:
		select {
		case totals := <-resp:
			foundation.Respond(w, http.StatusOK, totals)
		case <-ctx.Done():
			foundation.Respond(w, http.StatusRequestTimeout, newErrResp(http.StatusText(http.StatusRequestTimeout)))
			return
		}
	// NOTE: In a real system, you might want to implement backpressure or rate-limiting
	// to avoid overwhelming the event processing loop.
	// Here we just show how it could be done, returning a 429 Too Many Requests status,
	// we could also send a Retry-After header.
	case <-time.After(backpressureTimeout):
		foundation.Respond(w, http.StatusTooManyRequests, newErrResp("server busy, try again"))
		return
	case <-ctx.Done():
		foundation.Respond(w, http.StatusRequestTimeout, newErrResp(http.StatusText(http.StatusRequestTimeout)))
		return
	}
}
