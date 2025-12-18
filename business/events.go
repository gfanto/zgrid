package business

// Event represents any message processed by the grid loop.
type Event any

// GraphUpdate carries a new topology and an optional reply channel.
type GraphUpdate struct {
	Graph Graph
	Reply chan<- [][]string
}

// MeasurementUpdate carries a measurement and an optional reply channel.
type MeasurementUpdate struct {
	NodeMeasurement
	Reply chan<- []IslandMeasurement
}
