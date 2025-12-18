package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

var (
	version = "--- set from makefile ---"

	help        = flag.Bool("help", false, "show help message")
	showVersion = flag.Bool("version", false, "show command version")
	addr        = flag.String("addr", ":8000", "HTTP network address")
	nodeCount   = flag.Int("nodes", 100, "number of nodes in the topology graph")
	edgeCount   = flag.Int("edges", 150, "number of random edges in the topology graph")
	interval    = flag.Duration("interval", 20*time.Millisecond, "delay between measurement posts")
)

func main() {
	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	if *showVersion {
		fmt.Println(version)
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "application error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	if *nodeCount <= 0 {
		return fmt.Errorf("invalid -nodes: must be > 0")
	}
	if *edgeCount < 0 {
		return fmt.Errorf("invalid -edges: must be >= 0")
	}
	if *interval <= 0 {
		return fmt.Errorf("invalid -interval: must be > 0")
	}

	baseURL := buildBaseURL(*addr)
	httpClient := &http.Client{Timeout: 5 * time.Second}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	graph := buildRandomGraph(rng, *nodeCount, *edgeCount)
	if err := postJSON(ctx, httpClient, baseURL, "/graph", graph); err != nil {
		return fmt.Errorf("send graph: %w", err)
	}

	nodes := graph.Nodes
	t := time.NewTicker(*interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}

		p := MeasurementPayload{
			Node:  nodes[rng.Intn(len(nodes))],
			Value: rng.Float64() * 100,
		}

		if err := postJSON(ctx, httpClient, baseURL, "/measurements", p); err != nil {
			return fmt.Errorf("send measurement: %w", err)
		}
	}
}

type GraphPayload struct {
	Nodes []string   `json:"nodes"`
	Edges [][]string `json:"edges"`
}

type MeasurementPayload struct {
	Node  string  `json:"node"`
	Value float64 `json:"value"`
}

func postJSON(ctx context.Context, client *http.Client, baseURL, path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("POST %s: unexpected status %s", path, resp.Status)
	}

	return nil
}

func buildBaseURL(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "http://127.0.0.1:8000"
	}

	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr
	}
	if !strings.Contains(addr, ":") {
		addr += ":8000"
	}
	return "http://" + addr
}

func buildRandomGraph(rng *rand.Rand, nodeCount, edgeCount int) GraphPayload {
	nodes := make([]string, 0, nodeCount)
	for i := range nodeCount {
		nodes = append(nodes, fmt.Sprintf("N%d", i))
	}

	var edges [][]string
	for range edgeCount {
		a := nodes[rng.Intn(len(nodes))]
		b := nodes[rng.Intn(len(nodes))]
		if a != b {
			edges = append(edges, []string{a, b})
		}
	}

	return GraphPayload{Nodes: nodes, Edges: edges}
}
