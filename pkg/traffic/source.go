package traffic

import "context"

// TrafficSource interface - each traffic monitoring tool implements this.
type TrafficSource interface {
	// Name returns the source identifier (e.g., "hubble", "caretta")
	Name() string

	// Detect checks if this traffic source is available in the cluster
	Detect(ctx context.Context) (*DetectionResult, error)

	// GetFlows retrieves aggregated flow data from the source
	GetFlows(ctx context.Context, opts FlowOptions) (*FlowsResponse, error)

	// StreamFlows returns a channel of flows for real-time updates
	StreamFlows(ctx context.Context, opts FlowOptions) (<-chan Flow, error)

	// Close cleans up any resources (e.g., gRPC connections)
	Close() error
}
