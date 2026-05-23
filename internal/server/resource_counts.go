package server

import (
	"log"
	"net/http"
	"sync"

	"github.com/skyhook-io/radar/internal/k8s"
	"github.com/skyhook-io/radar/pkg/k8score"
)

type ResourceCountsResponse struct {
	Counts    map[string]int `json:"counts"`
	Forbidden []string       `json:"forbidden,omitempty"`
}

func (s *Server) handleResourceCounts(w http.ResponseWriter, r *http.Request) {
	if !s.requireConnected(w) {
		return
	}

	namespaces := s.parseNamespacesForUser(r)
	if noNamespaceAccess(namespaces) {
		s.writeJSON(w, ResourceCountsResponse{Counts: map[string]int{}})
		return
	}

	cache := k8s.GetResourceCache()
	if cache == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Resource cache not available")
		return
	}

	counts := make(map[string]int)
	var forbidden []string

	for _, kl := range k8score.AllKindListers() {
		l := kl.Lister()(cache.ResourceCache)
		if l == nil {
			forbidden = append(forbidden, kl.CountKey())
			continue
		}
		// Cluster-scoped kinds: ListCountNamespaced ignores the namespace
		// filter and returns the cluster-wide count, so authorize the kind
		// per-user via SAR before counting.
		if k8s.IsClusterOnlyKind(kl.Kind()) {
			group, resource, ok := k8s.ClusterOnlyKindGVR(kl.Kind())
			if !ok || !s.canRead(r, group, resource, "", "list") {
				continue
			}
		}
		n := k8score.ListCountNamespaced(l, namespaces)
		// Namespaces is cluster-scoped but exposed as a filtered list. For
		// namespace-restricted users (non-empty filter), the lister can't
		// honor the filter, so we report the count of namespaces they're
		// allowed to see rather than leaking the cluster-wide total.
		if kl.Kind() == "Namespace" && len(namespaces) > 0 {
			n = len(namespaces)
		}
		if n > 0 {
			counts[kl.CountKey()] = n
		}
	}

	// 2. Dynamic resources (CRDs) — counted concurrently since each Count() hits a separate informer indexer
	discovery := k8s.GetResourceDiscovery()
	dynamicCache := k8s.GetDynamicResourceCache()
	if discovery != nil && dynamicCache != nil {
		resources, err := discovery.GetAPIResources()
		if err != nil {
			log.Printf("[resource-counts] Failed to discover API resources for CRD counts: %v", err)
		} else {
			// Deduplicate CRDs by group+kind
			type crdInfo struct {
				kind       string
				group      string
				resource   string
				namespaced bool
			}
			seen := make(map[string]bool)
			var crds []crdInfo
			for _, res := range resources {
				if !res.IsCRD {
					continue
				}
				key := res.Group + "/" + res.Kind
				if !seen[key] {
					seen[key] = true
					crds = append(crds, crdInfo{kind: res.Kind, group: res.Group, resource: res.Name, namespaced: res.Namespaced})
				}
			}

			var mu sync.Mutex
			var wg sync.WaitGroup
			for _, crd := range crds {
				wg.Add(1)
				go func(c crdInfo) {
					defer wg.Done()
					gvr, ok := discovery.GetGVRWithGroup(c.kind, c.group)
					if !ok {
						return
					}
					ns := namespaces
					if !c.namespaced {
						// Cluster-scoped CRD: per-kind SAR gate before
						// listing cluster-wide. Mirrors the dashboard's
						// collectClusterScopedCRDCounts.
						if !s.canRead(r, c.group, c.resource, "", "list") {
							return
						}
						ns = nil
					}
					n, err := dynamicCache.Count(gvr, ns)
					if err != nil {
						log.Printf("[resource-counts] Failed to count CRD %s/%s: %v", c.group, c.kind, err)
						return
					}
					if n == 0 {
						return
					}
					countKey := c.group + "/" + c.kind
					mu.Lock()
					counts[countKey] = n
					mu.Unlock()
				}(crd)
			}
			wg.Wait()
		}
	}

	s.writeJSON(w, ResourceCountsResponse{
		Counts:    counts,
		Forbidden: forbidden,
	})
}
