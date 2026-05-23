package prometheus

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAnyNodeUsesDocker(t *testing.T) {
	tests := []struct {
		name    string
		nodes   []*corev1.Node
		wantHit bool
	}{
		{
			name:    "no nodes",
			nodes:   nil,
			wantHit: false,
		},
		{
			name: "containerd only",
			nodes: []*corev1.Node{
				nodeWithRuntime("node-1", "containerd://1.6.21"),
				nodeWithRuntime("node-2", "containerd://1.7.0"),
			},
			wantHit: false,
		},
		{
			name: "cri-o only",
			nodes: []*corev1.Node{
				nodeWithRuntime("node-1", "cri-o://1.27.1"),
			},
			wantHit: false,
		},
		{
			name: "docker only",
			nodes: []*corev1.Node{
				nodeWithRuntime("node-1", "docker://24.0.2"),
			},
			wantHit: true,
		},
		{
			name: "mixed — one docker node among containerd",
			nodes: []*corev1.Node{
				nodeWithRuntime("node-1", "containerd://1.6.21"),
				nodeWithRuntime("node-2", "docker://20.10.23"),
				nodeWithRuntime("node-3", "containerd://1.7.0"),
			},
			wantHit: true,
		},
		{
			name: "empty runtime string",
			nodes: []*corev1.Node{
				nodeWithRuntime("node-1", ""),
			},
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := anyNodeUsesDocker(tt.nodes)
			if tt.wantHit && got == "" {
				t.Error("expected cri-docker hint, got empty string")
			}
			if !tt.wantHit && got != "" {
				t.Errorf("expected no hint, got: %s", got)
			}
			if tt.wantHit && got != criDockerHint {
				t.Errorf("hint text mismatch:\n  got:  %s\n  want: %s", got, criDockerHint)
			}
		})
	}
}

func nodeWithRuntime(name, runtimeVersion string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			NodeInfo: corev1.NodeSystemInfo{
				ContainerRuntimeVersion: runtimeVersion,
			},
		},
	}
}
