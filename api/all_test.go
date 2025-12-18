package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
	"zgrid/business"
	"zgrid/foundation"
)

func TestGraphEndpointSuccess(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	events := make(chan business.Event, 16)
	grid := business.NewGrid()
	go grid.Loop(ctx, events)
	t.Cleanup(func() { close(events) })

	h := foundation.WrapMiddleware(All(), GridEventsMiddleware(events))

	type graphPayload struct {
		Nodes []string   `json:"nodes"`
		Edges [][]string `json:"edges"`
	}

	var resp struct {
		Islands [][]string `json:"islands"`
	}
	status := postJSON(t, h, "/graph", graphPayload{
		Nodes: []string{"A", "B", "C", "D"},
		Edges: [][]string{{"A", "B"}, {"C", "D"}},
	}, &resp)
	if status != 200 {
		t.Fatalf("status = %d, want %d", status, 200)
	}

	want := [][]string{{"A", "B"}, {"C", "D"}}
	if !islandsEqual(resp.Islands, want) {
		t.Fatalf("islands = %v, want %v", resp.Islands, want)
	}
}

func TestMeasurementsEndpointSuccess(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	events := make(chan business.Event, 64)
	grid := business.NewGrid()
	go grid.Loop(ctx, events)
	t.Cleanup(func() { close(events) })

	h := foundation.WrapMiddleware(All(), GridEventsMiddleware(events))

	type graphPayload struct {
		Nodes []string   `json:"nodes"`
		Edges [][]string `json:"edges"`
	}
	postJSON(t, h, "/graph", graphPayload{
		Nodes: []string{"A", "B", "C", "D"},
		Edges: [][]string{{"A", "B"}, {"C", "D"}},
	}, nil)

	type measurementPayload struct {
		Node  string  `json:"node"`
		Value float64 `json:"value"`
	}

	var totals []business.IslandMeasurement
	status := postJSON(t, h, "/measurements", measurementPayload{Node: "A", Value: 5.3}, &totals)
	if status != 200 {
		t.Fatalf("status = %d, want %d", status, 200)
	}

	if len(totals) != 2 {
		t.Fatalf("len(totals) = %d, want %d", len(totals), 2)
	}
	if got := totals[0].Total; !floatEqual(got, 5.3) {
		t.Fatalf("totals[0].Total = %v, want %v", got, 5.3)
	}
}

func TestTopologyChangeRetainsMeasurements(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	events := make(chan business.Event, 256)
	grid := business.NewGrid()
	go grid.Loop(ctx, events)
	t.Cleanup(func() { close(events) })

	h := foundation.WrapMiddleware(All(), GridEventsMiddleware(events))

	type graphPayload struct {
		Nodes []string   `json:"nodes"`
		Edges [][]string `json:"edges"`
	}
	type measurementPayload struct {
		Node  string  `json:"node"`
		Value float64 `json:"value"`
	}

	// Graph 1: [A B] and [C D]
	postJSON(t, h, "/graph", graphPayload{
		Nodes: []string{"A", "B", "C", "D"},
		Edges: [][]string{{"A", "B"}, {"C", "D"}},
	}, nil)

	var got []business.IslandMeasurement
	postJSON(t, h, "/measurements", measurementPayload{Node: "A", Value: 5.3}, &got)
	postJSON(t, h, "/measurements", measurementPayload{Node: "B", Value: 10.1}, &got)

	if len(got) != 2 {
		t.Fatalf("len(totals) = %d, want %d", len(got), 2)
	}
	if !floatEqual(got[0].Total, 15.4) {
		t.Fatalf("graph1 island0 total = %v, want %v", got[0].Total, 15.4)
	}
	if !floatEqual(got[1].Total, 0) {
		t.Fatalf("graph1 island1 total = %v, want %v", got[1].Total, 0.0)
	}

	// Graph 2: [A C] and [B D]
	postJSON(t, h, "/graph", graphPayload{
		Nodes: []string{"A", "B", "C", "D"},
		Edges: [][]string{{"A", "C"}, {"B", "D"}},
	}, nil)

	postJSON(t, h, "/measurements", measurementPayload{Node: "C", Value: 1.1}, &got)
	if len(got) != 2 {
		t.Fatalf("len(totals) = %d, want %d", len(got), 2)
	}
	if !floatEqual(got[0].Total, 6.4) {
		t.Fatalf("graph2 island0 total = %v, want %v", got[0].Total, 6.4)
	}
	if !floatEqual(got[1].Total, 10.1) {
		t.Fatalf("graph2 island1 total = %v, want %v", got[1].Total, 10.1)
	}
}

func TestGraphEndpointErrors(t *testing.T) {
	t.Parallel()

	type graphPayload struct {
		Nodes []string   `json:"nodes"`
		Edges [][]string `json:"edges"`
	}

	tests := []struct {
		name        string
		handler     http.Handler
		method      string
		contentType string
		body        any
		wantStatus  int
	}{
		{
			name:        "missing middleware returns 500",
			handler:     All(),
			method:      http.MethodPost,
			contentType: "application/json",
			body: graphPayload{
				Nodes: []string{"A"},
				Edges: [][]string{},
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:        "wrong method returns 405",
			handler:     foundation.WrapMiddleware(All(), GridEventsMiddleware(make(chan business.Event))),
			method:      http.MethodGet,
			contentType: "application/json",
			body:        nil,
			wantStatus:  http.StatusMethodNotAllowed,
		},
		{
			name:        "wrong content-type returns 415",
			handler:     foundation.WrapMiddleware(All(), GridEventsMiddleware(make(chan business.Event))),
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "nope",
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "invalid edge shape returns 400",
			handler:     foundation.WrapMiddleware(All(), GridEventsMiddleware(make(chan business.Event))),
			method:      http.MethodPost,
			contentType: "application/json",
			body: map[string]any{
				"nodes": []string{"A", "B"},
				"edges": [][]string{{"A"}},
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := doRequest(t, tt.handler, tt.method, "/graph", tt.contentType, tt.body)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status, tt.wantStatus)
			}
		})
	}
}

func TestGraphEndpointInvalidPayloadReturnsJSONError(t *testing.T) {
	t.Parallel()

	h := foundation.WrapMiddleware(All(), GridEventsMiddleware(make(chan business.Event)))

	req := httptest.NewRequest(http.MethodPost, "http://example.test/graph", bytes.NewReader([]byte(`{"nodes":["A","B"],"edges":[["A"]]}`)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", ct, "application/json")
	}

	var got struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Error == "" {
		t.Fatalf("error message is empty")
	}
}

func TestBackpressureReturns429WhenNoConsumer(t *testing.T) {
	t.Parallel()

	events := make(chan business.Event) // unbuffered, no consumer => send blocks
	h := foundation.WrapMiddleware(All(), GridEventsMiddleware(events))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	t.Cleanup(cancel)

	status := doRequestWithContext(t, ctx, h, "POST", "/graph", "application/json", map[string]any{
		"nodes": []string{"A", "B"},
		"edges": [][]string{{"A", "B"}},
	})
	if status != http.StatusRequestTimeout {
		t.Fatalf("status = %d, want %d", status, http.StatusRequestTimeout)
	}
}

func TestMeasurementsBurstDoesNotDeadlock(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	events := make(chan business.Event, 1024)
	grid := business.NewGrid()
	go grid.Loop(ctx, events)
	t.Cleanup(func() { close(events) })

	h := foundation.WrapMiddleware(All(), GridEventsMiddleware(events))

	postJSON(t, h, "/graph", map[string]any{
		"nodes": []string{"A", "B", "C", "D"},
		"edges": [][]string{{"A", "B"}, {"C", "D"}},
	}, nil)

	type measurementPayload struct {
		Node  string  `json:"node"`
		Value float64 `json:"value"`
	}

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			node := "A"
			if i%2 == 1 {
				node = "C"
			}
			var out []business.IslandMeasurement
			status := postJSON(t, h, "/measurements", measurementPayload{
				Node:  node,
				Value: float64(i),
			}, &out)
			if status != http.StatusOK {
				t.Errorf("status = %d, want %d", status, http.StatusOK)
			}
		}(i)
	}
	wg.Wait()
}

func postJSON(t *testing.T, h http.Handler, path string, payload any, out any) int {
	t.Helper()

	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest("POST", "http://example.test"+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if out == nil {
		return rr.Code
	}

	if err := json.NewDecoder(rr.Body).Decode(out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return rr.Code
}

func doRequest(t *testing.T, h http.Handler, method, path, contentType string, body any) int {
	t.Helper()

	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, "http://example.test"+path, r)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code
}

func doRequestWithContext(t *testing.T, ctx context.Context, h http.Handler, method, path, contentType string, body any) int {
	t.Helper()

	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}

	req := httptest.NewRequestWithContext(ctx, method, "http://example.test"+path, r)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code
}

func islandsEqual(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}

func floatEqual(a, b float64) bool {
	const eps = 1e-9
	return math.Abs(a-b) <= eps
}
