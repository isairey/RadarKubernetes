package k8score

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DefaultDebugImage is the default image for ephemeral debug containers.
const DefaultDebugImage = "busybox:latest"

// EphemeralContainerOptions configures debug container creation.
type EphemeralContainerOptions struct {
	Namespace       string
	PodName         string
	TargetContainer string // Container to share process namespace with
	Image           string // Debug image (default: busybox:latest)
	ContainerName   string // Name for ephemeral container (auto-generated if empty)
}

// CreateEphemeralContainer adds an ephemeral debug container to a pod.
func CreateEphemeralContainer(ctx context.Context, client kubernetes.Interface, opts EphemeralContainerOptions) (*corev1.EphemeralContainer, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}
	if opts.Image == "" {
		opts.Image = DefaultDebugImage
	}
	if opts.ContainerName == "" {
		opts.ContainerName = fmt.Sprintf("debug-%d", time.Now().Unix())
	}

	pod, err := client.CoreV1().Pods(opts.Namespace).Get(ctx, opts.PodName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	ec := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:                     opts.ContainerName,
			Image:                    opts.Image,
			ImagePullPolicy:          corev1.PullIfNotPresent,
			Stdin:                    true,
			TTY:                      true,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		},
		TargetContainerName: opts.TargetContainer,
	}

	pod.Spec.EphemeralContainers = append(pod.Spec.EphemeralContainers, ec)

	_, err = client.CoreV1().Pods(opts.Namespace).UpdateEphemeralContainers(
		ctx,
		opts.PodName,
		pod,
		metav1.UpdateOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ephemeral container: %w", err)
	}

	return &ec, nil
}

// WaitForEphemeralContainer polls until an ephemeral container reaches Running state or timeout.
func WaitForEphemeralContainer(ctx context.Context, client kubernetes.Interface, namespace, podName, containerName string, timeout time.Duration) error {
	if client == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get pod: %w", err)
		}

		for _, status := range pod.Status.EphemeralContainerStatuses {
			if status.Name == containerName {
				if status.State.Running != nil {
					return nil
				}
				if status.State.Terminated != nil {
					return fmt.Errorf("container terminated: %s", status.State.Terminated.Reason)
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			continue
		}
	}

	return fmt.Errorf("timeout waiting for ephemeral container to start")
}
