package business

import "context"

// Grid stores the current topology (graph/islands) and the latest measurement per node.
//
// Grid is not safe for concurrent access; mutate it only via Loop, which processes
// events serially.
//
// Measurements are retained across topology updates. Aggregation sums only nodes
// present in the current graph (via nodeToIsland), so measurements for absent nodes
// do not affect totals.
type Grid struct {
	graph        Graph              // current grid graph
	islands      [][]string         // list of islands (each island is a list of nodes)
	nodeToIsland map[string]int     // node -> island index
	measurements map[string]float64 // node -> latest measurement
}

// NewGrid initializes an empty grid state.
func NewGrid() *Grid {
	return &Grid{
		graph:        Graph{Nodes: []string{}, Edges: map[string][]string{}},
		islands:      [][]string{},
		nodeToIsland: map[string]int{},
		measurements: map[string]float64{},
	}
}

// Loop processes graph and measurement events until the channel closes.
func (s *Grid) Loop(ctx context.Context, evts <-chan Event) {
	// Process events serially to avoid concurrency issues.
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-evts:
			if !ok {
				return
			}
			s.update(e)
		}
	}
}

func (s *Grid) update(evt Event) {
	// If the graph is updated, the islands are recomputed, and the measurements
	// remain, since they are per-node.
	// If a measurement is updated, the totals per island are recomputed each
	// time.
	// Each event may have an optional reply channel to send back results.
	switch e := evt.(type) {
	case GraphUpdate:
		s.graph = e.Graph
		// Measurements for nodes not in the new graph are retained but ignored
		// during aggregation. This allows the grid to be dynamic without losing
		// data for nodes that may reappear later.
		s.islands, s.nodeToIsland = computeIslands(s.graph)
		if e.Reply != nil {
			e.Reply <- s.islands
		}
	case MeasurementUpdate:
		// Update measurement only if the node exists in the current graph.
		// This avoids storing measurements for nodes that are not part of the grid.
		// Sending measurements for non-existent nodes is allowed.
		if s.graph.HasNode(e.Node) {
			s.measurements[e.Node] = e.Value
		}

		totals := aggregate(s)
		if e.Reply != nil {
			e.Reply <- totals
		}
	}
}

// computeIslands walks the graph and returns the connected components along with
// a reverse index from node name to island position. Islands are discovered via
// an iterative DFS to avoid recursion limits.
func computeIslands(g Graph) ([][]string, map[string]int) {
	visited := map[string]bool{}
	var islands [][]string
	nodeToIsland := map[string]int{}

	for _, n := range g.Nodes {
		if visited[n] {
			continue
		}
		stack := []string{n}
		var island []string

		for len(stack) > 0 {
			v := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if visited[v] {
				continue
			}
			visited[v] = true
			island = append(island, v)
			// Push unvisited neighbors so they are explored in this component.
			for _, nei := range g.Edges[v] {
				if !visited[nei] {
					stack = append(stack, nei)
				}
			}
		}

		idx := len(islands)
		// Map each node in the new island to its island index for O(1) lookups.
		for _, node := range island {
			nodeToIsland[node] = idx
		}
		islands = append(islands, island)
	}
	return islands, nodeToIsland
}

// aggregate sums the latest measurement for each node into its island and
// returns one IslandMeasurement entry per island in the current graph.
func aggregate(s *Grid) []IslandMeasurement {
	totals := make([]float64, len(s.islands))

	for node, val := range s.measurements {
		// Ignore stale measurements from nodes not present in the current graph.
		if idx, ok := s.nodeToIsland[node]; ok {
			totals[idx] += val
		}
	}

	res := make([]IslandMeasurement, len(s.islands))
	for i, island := range s.islands {
		res[i] = IslandMeasurement{
			Island: island,
			Total:  totals[i],
		}
	}

	return res
}
