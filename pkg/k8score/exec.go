package k8score

import (
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// NewPodExecExecutor creates an SPDY executor for running commands in a pod container.
// The caller uses the returned Executor to call StreamWithContext.
func NewPodExecExecutor(client kubernetes.Interface, config *rest.Config, namespace, podName, containerName string, command []string, tty bool) (remotecommand.Executor, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}
	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			// When TTY is true, the terminal muxes stderr into stdout, so Stderr must be false.
			// Setting both TTY and Stderr causes SPDY stream errors on some API servers; matches kubectl exec -it.
			Stderr: !tty,
			TTY:    tty,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, http.MethodPost, req.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to create exec executor for %s/%s/%s: %w", namespace, podName, containerName, err)
	}

	return executor, nil
}
