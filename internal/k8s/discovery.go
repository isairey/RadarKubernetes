package k8s

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/skyhook-io/radar/pkg/k8score"
)

// ResourceDiscovery wraps the shared k8score implementation.
// The singleton pattern (Init/Get/Reset) stays here because it depends on
// the package-level GetDiscoveryClient() function.
type ResourceDiscovery struct {
	*k8score.ResourceDiscovery
}

// Re-export types so existing callers compile without changes.
type APIResource = k8score.APIResource
type DiscoveryStats = k8score.DiscoveryStats

var (
	resourceDiscovery *ResourceDiscovery
	discoveryOnce     = new(sync.Once)
	discoveryMu       sync.Mutex
)

// isMoreStableVersion delegates to the shared implementation.
// Needed by dynamic_cache.go (same package, calls it without qualifier).
var isMoreStableVersion = k8score.IsMoreStableVersion

// InitResourceDiscovery initializes the resource discovery module.
func InitResourceDiscovery() error {
	var initErr error
	discoveryOnce.Do(func() {
		client := GetDiscoveryClient()
		core, err := k8score.NewResourceDiscovery(client)
		if err != nil {
			initErr = err
			return
		}
		resourceDiscovery = &ResourceDiscovery{ResourceDiscovery: core}
	})
	return initErr
}

// GetResourceDiscovery returns the singleton discovery instance.
func GetResourceDiscovery() *ResourceDiscovery {
	return resourceDiscovery
}

// ResetResourceDiscovery clears the resource discovery instance so it can be
// reinitialized for a new cluster after context switch.
func ResetResourceDiscovery() {
	discoveryMu.Lock()
	defer discoveryMu.Unlock()

	resourceDiscovery = nil
	discoveryOnce = new(sync.Once)
}

// Refresh delegates to the embedded implementation.
// Note: GetAPIResources, GetGVR, GetGVRWithGroup, GetResource, IsKnownResource,
// IsCRD, SupportsWatch, SupportsWatchGVR, GetKindForGVR, and Stats are all
// promoted from the embedded *k8score.ResourceDiscovery and work without wrappers.

// GetGVR returns the GroupVersionResource — needed as a direct method because
// the embedded type is accessed via pointer and callers pass *ResourceDiscovery.
func (d *ResourceDiscovery) GetGVR(kindOrName string) (schema.GroupVersionResource, bool) {
	if d == nil || d.ResourceDiscovery == nil {
		return schema.GroupVersionResource{}, false
	}
	return d.ResourceDiscovery.GetGVR(kindOrName)
}

// GetGVRWithGroup returns the GroupVersionResource for a kind with a specific API group.
func (d *ResourceDiscovery) GetGVRWithGroup(kindOrName string, group string) (schema.GroupVersionResource, bool) {
	if d == nil || d.ResourceDiscovery == nil {
		return schema.GroupVersionResource{}, false
	}
	return d.ResourceDiscovery.GetGVRWithGroup(kindOrName, group)
}

// SupportsWatchGVR checks if a GVR supports list and watch verbs.
func (d *ResourceDiscovery) SupportsWatchGVR(gvr schema.GroupVersionResource) bool {
	if d == nil || d.ResourceDiscovery == nil {
		return false
	}
	return d.ResourceDiscovery.SupportsWatchGVR(gvr)
}

// GetKindForGVR returns the Kind name for a given GVR.
func (d *ResourceDiscovery) GetKindForGVR(gvr schema.GroupVersionResource) string {
	if d == nil || d.ResourceDiscovery == nil {
		return ""
	}
	return d.ResourceDiscovery.GetKindForGVR(gvr)
}
