package business

import (
	"reflect"
	"sort"
	"testing"
)

func TestNewGraph(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		nodes      []string
		edges      [][]string
		wantEdges  map[string][]string
		wantHasKey map[string]bool
	}{
		{
			name:  "undirected adjacency is built",
			nodes: []string{"A", "B"},
			edges: [][]string{{"A", "B"}},
			wantEdges: map[string][]string{
				"A": {"B"},
				"B": {"A"},
			},
		},
		{
			name:  "unknown nodes in edges are ignored",
			nodes: []string{"A"},
			edges: [][]string{{"A", "X"}, {"X", "A"}},
			wantEdges: map[string][]string{
				"A": {},
			},
			wantHasKey: map[string]bool{
				"X": false,
			},
		},
		{
			name:  "malformed edges do not panic and are skipped",
			nodes: []string{"A", "B"},
			edges: [][]string{{"A"}, {"A", "B", "C"}},
			wantEdges: map[string][]string{
				"A": {},
				"B": {},
			},
		},
		{
			name:  "multiple edges connect components",
			nodes: []string{"A", "B", "C"},
			edges: [][]string{{"A", "B"}, {"B", "C"}},
			wantEdges: map[string][]string{
				"A": {"B"},
				"B": {"A", "C"},
				"C": {"B"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGraph(tt.nodes, tt.edges)

			if !reflect.DeepEqual(g.Nodes, tt.nodes) {
				t.Fatalf("NewGraph() Nodes = %v, want %v", g.Nodes, tt.nodes)
			}

			for _, node := range tt.nodes {
				if _, ok := g.Edges[node]; !ok {
					t.Fatalf("NewGraph() Edges missing key for node %q", node)
				}
			}

			for node, wantNeighbors := range tt.wantEdges {
				gotNeighbors := normalizeStrings(g.Edges[node])
				wantNeighbors = normalizeStrings(wantNeighbors)
				if !reflect.DeepEqual(gotNeighbors, wantNeighbors) {
					t.Fatalf("NewGraph() Edges[%q] = %v, want %v", node, gotNeighbors, wantNeighbors)
				}
			}

			for node, wantPresent := range tt.wantHasKey {
				_, gotPresent := g.Edges[node]
				if gotPresent != wantPresent {
					t.Fatalf("NewGraph() map key %q present = %v, want %v", node, gotPresent, wantPresent)
				}
			}
		})
	}
}

func normalizeStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := append([]string(nil), in...)
	sort.Strings(out)
	n := 0
	for _, s := range out {
		if n == 0 || s != out[n-1] {
			out[n] = s
			n++
		}
	}
	return out[:n]
}
