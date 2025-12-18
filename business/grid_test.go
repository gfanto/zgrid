package business

import (
	"reflect"
	"testing"
)

func TestComputeIslands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		graph            Graph
		wantIslands      [][]string
		wantNodeToIsland map[string]int
	}{
		{
			name: "empty graph",
			graph: Graph{
				Nodes: []string{},
				Edges: map[string][]string{},
			},
			wantIslands:      [][]string{},
			wantNodeToIsland: map[string]int{},
		},
		{
			name: "single island",
			graph: Graph{
				Nodes: []string{"a", "b", "c"},
				Edges: map[string][]string{
					"a": {"b"},
					"b": {"a", "c"},
					"c": {"b"},
				},
			},
			wantIslands: [][]string{
				{"a", "b", "c"},
			},
			wantNodeToIsland: map[string]int{
				"a": 0,
				"b": 0,
				"c": 0,
			},
		},
		{
			name: "multiple islands",
			graph: Graph{
				Nodes: []string{"x", "y", "z", "w"},
				Edges: map[string][]string{
					"x": {"y"},
					"y": {"x"},
					"z": {"w"},
					"w": {"z"},
				},
			},
			wantIslands: [][]string{
				{"x", "y"},
				{"z", "w"},
			},
			wantNodeToIsland: map[string]int{
				"x": 0,
				"y": 0,
				"z": 1,
				"w": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIslands, gotNodeToIsland := computeIslands(tt.graph)
			if !islandsEqual(gotIslands, tt.wantIslands) {
				t.Fatalf("computeIslands() islands = %v, want %v", gotIslands, tt.wantIslands)
			}
			if !reflect.DeepEqual(gotNodeToIsland, tt.wantNodeToIsland) {
				t.Fatalf("computeIslands() nodeToIsland = %v, want %v", gotNodeToIsland, tt.wantNodeToIsland)
			}
		})
	}
}

func TestAggregate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		grid Grid
		want []IslandMeasurement
	}{
		{
			name: "sums measurements per island",
			grid: Grid{
				islands: [][]string{{"a", "b"}, {"c"}},
				nodeToIsland: map[string]int{
					"a": 0,
					"b": 0,
					"c": 1,
				},
				measurements: map[string]float64{
					"a": 1.5,
					"b": 2.5,
					"c": 10,
				},
			},
			want: []IslandMeasurement{
				{Island: []string{"a", "b"}, Total: 4},
				{Island: []string{"c"}, Total: 10},
			},
		},
		{
			name: "ignores unknown nodes",
			grid: Grid{
				islands: [][]string{{"a"}},
				nodeToIsland: map[string]int{
					"a": 0,
				},
				measurements: map[string]float64{
					"a":     5,
					"ghost": 9,
				},
			},
			want: []IslandMeasurement{
				{Island: []string{"a"}, Total: 5},
			},
		},
		{
			name: "no measurements yet",
			grid: Grid{
				islands: [][]string{{"solo"}},
				nodeToIsland: map[string]int{
					"solo": 0,
				},
				measurements: map[string]float64{},
			},
			want: []IslandMeasurement{
				{Island: []string{"solo"}, Total: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregate(&tt.grid)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("aggregate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGridUpdateHandlesEvents(t *testing.T) {
	t.Parallel()
	type measurementStep struct {
		measurement NodeMeasurement
		wantTotals  []IslandMeasurement
	}

	tests := []struct {
		name             string
		graph            Graph
		wantIslands      [][]string
		measurementSteps []measurementStep
	}{
		{
			name: "single island updates replace previous values",
			graph: Graph{
				Nodes: []string{"a", "b"},
				Edges: map[string][]string{
					"a": {"b"},
					"b": {"a"},
				},
			},
			wantIslands: [][]string{{"a", "b"}},
			measurementSteps: []measurementStep{
				{
					measurement: NodeMeasurement{Node: "a", Value: 1},
					wantTotals: []IslandMeasurement{
						{Island: []string{"a", "b"}, Total: 1},
					},
				},
				{
					measurement: NodeMeasurement{Node: "a", Value: 2},
					wantTotals: []IslandMeasurement{
						{Island: []string{"a", "b"}, Total: 2},
					},
				},
				{
					measurement: NodeMeasurement{Node: "b", Value: 3},
					wantTotals: []IslandMeasurement{
						{Island: []string{"a", "b"}, Total: 5},
					},
				},
			},
		},
		{
			name: "multiple islands and unknown nodes are ignored",
			graph: Graph{
				Nodes: []string{"a", "b", "c"},
				Edges: map[string][]string{
					"a": {"b"},
					"b": {"a"},
					"c": {},
				},
			},
			wantIslands: [][]string{{"a", "b"}, {"c"}},
			measurementSteps: []measurementStep{
				{
					measurement: NodeMeasurement{Node: "a", Value: 2.5},
					wantTotals: []IslandMeasurement{
						{Island: []string{"a", "b"}, Total: 2.5},
						{Island: []string{"c"}, Total: 0},
					},
				},
				{
					measurement: NodeMeasurement{Node: "ghost", Value: 10},
					wantTotals: []IslandMeasurement{
						{Island: []string{"a", "b"}, Total: 2.5},
						{Island: []string{"c"}, Total: 0},
					},
				},
				{
					measurement: NodeMeasurement{Node: "c", Value: 1.5},
					wantTotals: []IslandMeasurement{
						{Island: []string{"a", "b"}, Total: 2.5},
						{Island: []string{"c"}, Total: 1.5},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grid := NewGrid()
			graphReply := make(chan [][]string, 1)
			grid.update(GraphUpdate{Graph: tt.graph, Reply: graphReply})
			gotIslands := <-graphReply
			if !islandsEqual(gotIslands, tt.wantIslands) {
				t.Fatalf("graph update islands = %v, want %v", gotIslands, tt.wantIslands)
			}

			for i, step := range tt.measurementSteps {
				reply := make(chan []IslandMeasurement, 1)
				grid.update(MeasurementUpdate{NodeMeasurement: step.measurement, Reply: reply})
				gotTotals := <-reply
				if !reflect.DeepEqual(gotTotals, step.wantTotals) {
					t.Fatalf("measurement step %d totals = %v, want %v", i, gotTotals, step.wantTotals)
				}
			}
		})
	}
}

func islandsEqual(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}
