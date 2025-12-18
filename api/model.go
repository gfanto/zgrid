package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Edge represents a connection between two nodes in the graph.
type Edge []string

// MarshalJSON implements the json.Marshaler interface for Edge.
// Returns an error if the edge does not connect exactly two nodes.
func (e Edge) MarshalJSON() ([]byte, error) {
	if len(e) != 2 {
		return nil, fmt.Errorf("each edge must connect exactly two nodes")
	}

	data, err := json.Marshal([]string(e))
	if err != nil {
		return nil, err
	}

	return data, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for Edge.
// Returns an error if the edge does not connect exactly two nodes.
func (e *Edge) UnmarshalJSON(data []byte) error {
	if strings.Count(string(data), ",") != 1 {
		return fmt.Errorf("each edge must connect exactly two nodes")
	}

	var nodes []string
	if err := json.Unmarshal(data, &nodes); err != nil {
		return err
	}

	*e = Edge(nodes)
	return nil
}
