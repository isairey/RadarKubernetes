package k8s

import (
	"sync"

	"github.com/skyhook-io/radar/pkg/k8score"
	"k8s.io/client-go/kubernetes"
)

// InitTestResourceCache creates a resource cache from a fake or test client,
// bypassing RBAC checks and the normal Initialize/InitResourceCache flow.
// All resource types are enabled. Call ResetTestState to clean up.
//
// This is intended for integration tests only.
func InitTestResourceCache(client kubernetes.Interface) error {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	enabled := map[string]bool{
		"pods":                     true,
		"services":                 true,
		"deployments":              true,
		"daemonsets":               true,
		"statefulsets":             true,
		"replicasets":              true,
		"ingresses":                true,
		"configmaps":               true,
		"secrets":                  true,
		"events":                   true,
		"persistentvolumeclaims":   true,
		"nodes":                    true,
		"namespaces":               true,
		"jobs":                     true,
		"cronjobs":                 true,
		"horizontalpodautoscalers": true,
		"persistentvolumes":        true,
		"storageclasses":           true,
		"poddisruptionbudgets":     true,
		"roles":                    true,
		"clusterroles":             true,
		"rolebindings":             true,
		"clusterrolebindings":      true,
	}

	cfg := k8score.CacheConfig{
		Client:        client,
		ResourceTypes: enabled,
		// No deferred types for tests — all sync immediately
		DeferredTypes: map[string]bool{},
	}

	core, err := k8score.NewResourceCache(cfg)
	if err != nil {
		return err
	}

	initialSyncComplete = true

	resourceCache = &ResourceCache{
		ResourceCache:  core,
		secretsEnabled: true,
	}

	// Mark cacheOnce as "already executed" so InitResourceCache is a no-op.
	cacheOnce = new(sync.Once)
	cacheOnce.Do(func() {})

	return nil
}

// SetTestContextName is a test-only helper that overrides the package-level
// kubeconfig context name. Used by tests that exercise per-context state
// (e.g. namespace preferences) without needing to spin up a real client.
// Returns the previous value so callers can restore it on cleanup.
func SetTestContextName(name string) string {
	clientMu.Lock()
	prev := contextName
	contextName = name
	clientMu.Unlock()
	return prev
}

// ResetTestState tears down the resource cache and resets all package-level
// state so the next test starts clean.
//
// This is intended for integration tests only.
func ResetTestState() {
	// Reset resource cache
	ResetResourceCache()

	// Reset connection state
	connectionStatusMu.Lock()
	connectionStatus = ConnectionStatus{}
	connectionStatusMu.Unlock()

	// Reset connection callbacks
	connectionCallbacksMu.Lock()
	connectionCallbacks = nil
	connectionCallbacksMu.Unlock()

	// Reset capabilities cache
	capabilitiesMu.Lock()
	cachedCapabilities = nil
	capabilitiesMu.Unlock()

	// Reset resource permissions cache
	resourcePermsMu.Lock()
	cachedPermResult = nil
	resourcePermsMu.Unlock()

	// Reset operation context so stale cancellations don't leak between tests
	CancelOngoingOperations()
}
