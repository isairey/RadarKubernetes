package k8s

import (
	"context"
	"fmt"

	"github.com/skyhook-io/radar/pkg/k8score"
)

// Re-export types from pkg/k8score for backward compatibility with existing callers.
type PodMetrics = k8score.PodMetrics
type NodeMetrics = k8score.NodeMetrics
type MetricsMeta = k8score.MetricsMeta
type ContainerMetrics = k8score.ContainerMetrics
type ResourceUsage = k8score.ResourceUsage

// GetPodMetrics fetches metrics for a specific pod from the metrics.k8s.io API.
func GetPodMetrics(ctx context.Context, namespace, name string) (*PodMetrics, error) {
	client := GetDynamicClient()
	if client == nil {
		return nil, fmt.Errorf("dynamic client not initialized")
	}
	return k8score.GetPodMetrics(ctx, client, namespace, name)
}

// GetNodeMetrics fetches metrics for a specific node from the metrics.k8s.io API.
func GetNodeMetrics(ctx context.Context, name string) (*NodeMetrics, error) {
	client := GetDynamicClient()
	if client == nil {
		return nil, fmt.Errorf("dynamic client not initialized")
	}
	return k8score.GetNodeMetrics(ctx, client, name)
}
