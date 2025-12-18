package business

// NodeMeasurement represents a single measurement value reported by a node.
type NodeMeasurement struct {
	Node  string
	Value float64
}

// IslandMeasurement aggregates the sum of measurements for a connected island.
type IslandMeasurement struct {
	Island []string
	Total  float64
}

// Graph describes grid topology: a list of nodes and adjacency edges.
type Graph struct {
	Nodes []string
	Edges map[string][]string
}

// NewGraph creates graph from nodes and list of edges.
func NewGraph(nodes []string, edges [][]string) Graph {
	nodeSet := make(map[string]struct{}, len(nodes))
	for _, n := range nodes {
		nodeSet[n] = struct{}{}
	}

	graph := Graph{
		Nodes: nodes,
		Edges: make(map[string][]string),
	}
	// Initialize empty adjacency for all nodes to ensure stable map lookups.
	for _, n := range nodes {
		graph.Edges[n] = []string{}
	}
	// Build adjacency list
	for _, edge := range edges {
		if len(edge) != 2 {
			continue
		}
		a, b := edge[0], edge[1]
		if _, ok := nodeSet[a]; !ok {
			continue
		}
		if _, ok := nodeSet[b]; !ok {
			continue
		}
		graph.Edges[a] = append(graph.Edges[a], b)
		graph.Edges[b] = append(graph.Edges[b], a)
	}
	return graph
}

// HasNode reports whether the graph contains the given node.
func (g Graph) HasNode(node string) bool {
	if g.Edges == nil {
		return false
	}
	_, ok := g.Edges[node]
	return ok
}
