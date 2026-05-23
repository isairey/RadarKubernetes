package k8s

import (
	"context"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	authv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/skyhook-io/radar/pkg/k8score"
)

// ResourcePermissions indicates which resource types the user can list/watch
type ResourcePermissions struct {
	Pods                     bool `json:"pods"`
	Services                 bool `json:"services"`
	Deployments              bool `json:"deployments"`
	DaemonSets               bool `json:"daemonSets"`
	StatefulSets             bool `json:"statefulSets"`
	ReplicaSets              bool `json:"replicaSets"`
	Ingresses                bool `json:"ingresses"`
	ConfigMaps               bool `json:"configMaps"`
	Secrets                  bool `json:"secrets"`
	Events                   bool `json:"events"`
	PersistentVolumeClaims   bool `json:"persistentVolumeClaims"`
	Nodes                    bool `json:"nodes"`
	Namespaces               bool `json:"namespaces"`
	Jobs                     bool `json:"jobs"`
	CronJobs                 bool `json:"cronJobs"`
	HorizontalPodAutoscalers bool `json:"horizontalPodAutoscalers"`
	PersistentVolumes        bool `json:"persistentVolumes"`
	StorageClasses           bool `json:"storageClasses"`
	PodDisruptionBudgets     bool `json:"podDisruptionBudgets"`
	NetworkPolicies          bool `json:"networkPolicies"`
	ServiceAccounts          bool `json:"serviceAccounts"`
	Roles                    bool `json:"roles"`
	ClusterRoles             bool `json:"clusterRoles"`
	RoleBindings             bool `json:"roleBindings"`
	ClusterRoleBindings      bool `json:"clusterRoleBindings"`
	LimitRanges              bool `json:"limitRanges"`
	Gateways                 bool `json:"gateways"`
	HTTPRoutes               bool `json:"httpRoutes"`
}

// PermissionCheckResult holds the result of resource access probes.
//
// Two views over the same probe pass:
//   - Perms / NamespaceScoped / Namespace: a uniform projection (Perm=true if
//     any scope works; NamespaceScoped=true if at least one kind ended up
//     namespace-scoped). Used by callers that just want a "can the user see
//     anything?" answer.
//   - Scopes: the per-kind authoritative map that drives informer wiring —
//     some kinds may be cluster-wide while others are namespace-scoped on
//     the same cluster, which the uniform view cannot express.
type PermissionCheckResult struct {
	Perms           *ResourcePermissions
	NamespaceScoped bool   // True if at least one resource type ended up namespace-scoped
	Namespace       string // The fallback namespace used for namespace-scoped probes
	Scopes          map[string]k8score.ResourceScope
}

// Capabilities represents the features available based on RBAC permissions
type Capabilities struct {
	Exec          bool                 `json:"exec"`                  // Can create pods/exec (terminal feature)
	LocalTerminal bool                 `json:"localTerminal"`         // Local terminal available (not in-cluster, not disabled)
	Logs          bool                 `json:"logs"`                  // Can get pods/log (log viewer)
	PortForward   bool                 `json:"portForward"`           // Can create pods/portforward
	Secrets       bool                 `json:"secrets"`               // Can list secrets
	SecretsUpdate bool                 `json:"secretsUpdate"`         // Can update secrets (inline editing)
	HelmWrite     bool                 `json:"helmWrite"`             // Helm write ops (detected via secrets/create as sentinel RBAC check)
	NodeWrite     bool                 `json:"nodeWrite"`             // Can patch nodes (cordon/uncordon/drain)
	MCPEnabled    bool                 `json:"mcpEnabled"`            // MCP server is running
	Deployment    DeploymentInfo       `json:"deployment"`            // How / where this Radar binary is running. Tells the UI which chrome to render or suppress (e.g. embedded mode hides the cluster headline + local-MCP card because the hub already renders both).
	AuthEnabled   bool                 `json:"authEnabled,omitempty"` // Auth is enabled on the server
	Username      string               `json:"username,omitempty"`    // Authenticated username (when auth enabled)
	Resources     *ResourcePermissions `json:"resources,omitempty"`   // Per-resource-type permissions
}

// NamespaceCapabilities holds the effective exec/logs/portForward capabilities
// for a specific namespace. When global checks deny these capabilities,
// namespace-scoped RBAC re-checks may grant them.
type NamespaceCapabilities struct {
	Exec        bool `json:"exec"`
	Logs        bool `json:"logs"`
	PortForward bool `json:"portForward"`
}

// DeploymentInfo describes how / where this Radar binary is running.
// The frontend uses Mode to gate chrome that only makes sense in some
// topologies — e.g. cloud-connected mode hides the cluster headline
// because Radar Cloud's hub renders it in the top bar; in-cluster mode
// falls back to the platform label for the cluster name because the
// kubeconfig context is the meaningless "in-cluster" sentinel.
//
// The set is closed; if a new topology ships (air-gapped, on-prem-SAML,
// BYOC, ...), add a member here and update consumers — the bool
// alternative would force every consumer to grow `mode-A || mode-B`
// disjunctions ad hoc.
type DeploymentMode string

const (
	// DeploymentModeLocal: Radar binary running on a developer's
	// machine with a kubeconfig. The most common OSS path.
	DeploymentModeLocal DeploymentMode = "local"
	// DeploymentModeInCluster: Radar pod running inside the cluster
	// it's observing, with no kubeconfig. The kubeconfig context name
	// is set to the literal "in-cluster" sentinel during bootstrap.
	// Frontend should fall back to the platform label for headlines.
	DeploymentModeInCluster DeploymentMode = "in-cluster"
	// DeploymentModeCloud: Radar pod running in-cluster AND tunneled
	// to Radar Cloud's hub (RADAR_CLOUD_MODE=true; technically a
	// superset of in-cluster mode plus the outbound tunnel). The hub
	// shell renders cluster identity + MCP discovery surfaces, so the
	// embedded UI suppresses both.
	DeploymentModeCloud DeploymentMode = "cloud"
)

// DeploymentInfo is exposed in the Capabilities response. Currently
// just Mode; reserved as a struct so future deployment-scoped facts
// (region, cluster id surface, helm chart version) can be added
// without another wire-shape change.
type DeploymentInfo struct {
	Mode DeploymentMode `json:"mode"`
}

var (
	cachedCapabilities   *Capabilities
	capabilitiesMu       sync.RWMutex
	capabilitiesExpiry   time.Time
	capabilitiesTTL      = 60 * time.Second
	capabilitiesErrorTTL = 5 * time.Second // Short TTL when API errors caused fail-closed results

	// Per-namespace capability cache for lazy RBAC re-checks.
	// When global checks (cluster-wide + effective-namespace) deny
	// exec/logs/portForward, callers can re-check for a specific namespace.
	nsCapCache map[string]*nsCapEntry
	nsCapMu    sync.RWMutex

	// ForceDisableHelmWrite overrides the helmWrite capability to false (for dev testing)
	ForceDisableHelmWrite bool
	// ForceDisableExec overrides the exec capability to false (for dev testing)
	ForceDisableExec bool
	// ForceDisableLocalTerminal overrides the localTerminal capability to false (for dev testing)
	ForceDisableLocalTerminal bool
)

type nsCapEntry struct {
	caps   NamespaceCapabilities
	expiry time.Time
}

// CheckCapabilities checks RBAC permissions using SelfSubjectAccessReview.
// Results are cached for 60 seconds normally, or 5 seconds when API errors
// caused fail-closed results (to allow rapid retry without long UI disruption).
func CheckCapabilities(ctx context.Context) (*Capabilities, error) {
	capabilitiesMu.RLock()
	if cachedCapabilities != nil && time.Now().Before(capabilitiesExpiry) {
		caps := *cachedCapabilities
		capabilitiesMu.RUnlock()
		return &caps, nil
	}
	capabilitiesMu.RUnlock()

	// Compute capabilities WITHOUT holding the write lock.
	// Multiple concurrent callers may race, but redundant checks are harmless.
	// Critical: holding the lock during network calls blocks
	// InvalidateCapabilitiesCache() during context switch.

	if GetClient() == nil {
		// Return all false if client not initialized (fail closed)
		log.Printf("Warning: K8s client not initialized, returning restricted capabilities")
		return &Capabilities{Exec: false, Logs: false, PortForward: false, Secrets: false, SecretsUpdate: false, HelmWrite: false}, nil
	}

	// Don't start RBAC checks when disconnected — the exec credential plugin
	// serializes all API calls per-process, so browser-polled capability checks
	// would block retry/context-switch connectivity tests.
	if GetConnectionStatus().State == StateDisconnected {
		return &Capabilities{}, nil
	}

	// Use the operation context so RBAC checks are canceled on context switch.
	// This prevents stale exec plugin calls from serializing and blocking the
	// new context's connectivity test.
	checkCtx, cancel := NewOperationContext(10 * time.Second)
	defer cancel()

	capStart := time.Now()
	logTiming("   [caps] CheckCapabilities starting RBAC checks")

	// Check each capability in parallel.
	// Try cluster-wide first, then namespace-scoped as fallback for namespace-scoped users.
	// Track API errors to avoid caching transient failures for the full TTL.
	fallbackNs := GetEffectiveNamespace()
	var hadErrors atomic.Bool

	type capCheck struct {
		resource string
		verb     string
		result   *bool
	}

	caps := &Capabilities{}
	checks := []capCheck{
		{"pods/exec", "create", &caps.Exec},
		{"pods/log", "get", &caps.Logs},
		{"pods/portforward", "create", &caps.PortForward},
		{"secrets", "list", &caps.Secrets},
		{"secrets", "update", &caps.SecretsUpdate},
		{"secrets", "create", &caps.HelmWrite},
		{"nodes", "patch", &caps.NodeWrite},
	}

	var wg sync.WaitGroup
	wg.Add(len(checks))

	for _, check := range checks {
		go func(c capCheck) {
			defer wg.Done()
			allowed, apiErr := canI(checkCtx, "", "", c.resource, c.verb)
			if allowed {
				*c.result = true
				return
			}
			if fallbackNs != "" {
				allowed, nsApiErr := canI(checkCtx, fallbackNs, "", c.resource, c.verb)
				if allowed {
					*c.result = true
					return
				}
				apiErr = apiErr || nsApiErr
			}
			if apiErr {
				hadErrors.Store(true)
			}
		}(check)
	}

	wg.Wait()
	logTiming("   [caps] CheckCapabilities RBAC checks done (%v)", time.Since(capStart))

	// Local terminal is not RBAC-gated — it depends on runtime mode only
	caps.LocalTerminal = !IsInCluster() && !ForceDisableLocalTerminal

	if ForceDisableHelmWrite {
		caps.HelmWrite = false
	}
	if ForceDisableExec {
		caps.Exec = false
	}

	// Cache the result. Use a short TTL if API errors caused fail-closed results,
	// so transient K8s API failures don't hide UI controls for a full minute.
	ttl := capabilitiesTTL
	if hadErrors.Load() {
		ttl = capabilitiesErrorTTL
		log.Printf("Warning: capability checks had API errors, using short cache TTL (%v)", ttl)
	}
	capabilitiesMu.Lock()
	cachedCapabilities = caps
	capabilitiesExpiry = time.Now().Add(ttl)
	capabilitiesMu.Unlock()

	return caps, nil
}

// canI checks if the current user/service account can perform an action.
// Returns (allowed, apiErr) — wraps k8score.CanI with the singleton client.
func canI(ctx context.Context, namespace, group, resource, verb string) (allowed bool, apiErr bool) {
	if ctx.Err() != nil {
		logTiming("   [caps] canI(%s %s) skipped: context canceled", verb, resource)
		return false, true
	}
	return k8score.CanI(ctx, GetClient(), namespace, group, resource, verb)
}

// GetCachedCapabilities returns the cached capabilities without triggering
// RBAC checks. Returns nil if no cached result is available.
func GetCachedCapabilities() *Capabilities {
	capabilitiesMu.RLock()
	defer capabilitiesMu.RUnlock()
	if cachedCapabilities == nil {
		return nil
	}
	caps := *cachedCapabilities
	return &caps
}

// InvalidateCapabilitiesCache forces the next CheckCapabilities call to refresh
func InvalidateCapabilitiesCache() {
	capabilitiesMu.Lock()
	cachedCapabilities = nil
	capabilitiesMu.Unlock()

	// Also clear namespace-scoped cache
	nsCapMu.Lock()
	nsCapCache = nil
	nsCapMu.Unlock()
}

// CheckNamespaceCapabilities performs namespace-scoped RBAC checks for capabilities
// that were denied by global checks (cluster-wide + effective-namespace fallback).
// This enables lazy re-checking when a user views a resource in a specific namespace —
// they may have namespace-scoped RoleBindings that grant exec/logs/portForward in
// namespaces other than the kubeconfig default.
//
// Returns nil if no namespace-scoped re-check is needed (all capabilities already allowed).
func CheckNamespaceCapabilities(ctx context.Context, namespace string, globalCaps *Capabilities) (*NamespaceCapabilities, error) {
	if namespace == "" {
		return nil, nil
	}

	// If all three are already allowed globally, no need for namespace check
	if globalCaps.Exec && globalCaps.Logs && globalCaps.PortForward {
		return nil, nil
	}

	// Check namespace cache
	nsCapMu.RLock()
	if nsCapCache != nil {
		if entry, ok := nsCapCache[namespace]; ok && time.Now().Before(entry.expiry) {
			result := entry.caps
			nsCapMu.RUnlock()
			return &result, nil
		}
	}
	nsCapMu.RUnlock()

	if GetClient() == nil {
		return nil, nil // No override — caller will use global caps
	}

	checkCtx, cancel := NewOperationContext(10 * time.Second)
	defer cancel()

	result := &NamespaceCapabilities{
		Exec:        globalCaps.Exec,
		Logs:        globalCaps.Logs,
		PortForward: globalCaps.PortForward,
	}

	// Only re-check capabilities that were denied globally
	type capCheck struct {
		resource string
		verb     string
		result   *bool
	}

	var checks []capCheck
	if !globalCaps.Exec && !ForceDisableExec {
		checks = append(checks, capCheck{"pods/exec", "create", &result.Exec})
	}
	if !globalCaps.Logs {
		checks = append(checks, capCheck{"pods/log", "get", &result.Logs})
	}
	if !globalCaps.PortForward {
		checks = append(checks, capCheck{"pods/portforward", "create", &result.PortForward})
	}

	if len(checks) == 0 {
		return result, nil
	}

	var hadErrors atomic.Bool
	var wg sync.WaitGroup
	wg.Add(len(checks))
	for _, check := range checks {
		go func(c capCheck) {
			defer wg.Done()
			allowed, apiErr := canI(checkCtx, namespace, "", c.resource, c.verb)
			if allowed {
				*c.result = true
			}
			if apiErr {
				hadErrors.Store(true)
			}
		}(check)
	}
	wg.Wait()

	// Cache the result. Use short TTL when API errors caused fail-closed results,
	// matching the pattern in CheckCapabilities.
	ttl := capabilitiesTTL
	if hadErrors.Load() {
		ttl = capabilitiesErrorTTL
		log.Printf("Warning: namespace %s capability checks had API errors, using short cache TTL (%v)", namespace, ttl)
	}
	nsCapMu.Lock()
	if nsCapCache == nil {
		nsCapCache = make(map[string]*nsCapEntry)
	}
	nsCapCache[namespace] = &nsCapEntry{
		caps:   *result,
		expiry: time.Now().Add(ttl),
	}
	nsCapMu.Unlock()

	return result, nil
}

// Per-user capabilities cache (keyed by username)
var (
	userCapabilitiesCache sync.Map // map[string]*userCapEntry
	userCapabilitiesTTL   = 60 * time.Second
)

type userCapEntry struct {
	caps      *Capabilities
	expiresAt time.Time
}

// CheckCapabilitiesForUser runs SubjectAccessReview as the given user
// to determine what the user can do (exec, logs, delete, helm, etc.)
// Results are cached per-user with 60s TTL.
func CheckCapabilitiesForUser(ctx context.Context, username string, groups []string) (*Capabilities, error) {
	// Check cache
	if entry, ok := userCapabilitiesCache.Load(username); ok {
		e := entry.(*userCapEntry)
		if time.Now().Before(e.expiresAt) {
			caps := *e.caps
			return &caps, nil
		}
	}

	k8sClient := GetClient()
	if k8sClient == nil {
		return &Capabilities{}, nil
	}

	if GetConnectionStatus().State == StateDisconnected {
		return &Capabilities{}, nil
	}

	checkCtx, cancel := NewOperationContext(10 * time.Second)
	defer cancel()

	type capCheck struct {
		resource string
		verb     string
		result   *bool
	}

	caps := &Capabilities{}
	checks := []capCheck{
		{"pods/exec", "create", &caps.Exec},
		{"pods/log", "get", &caps.Logs},
		{"pods/portforward", "create", &caps.PortForward},
		{"secrets", "list", &caps.Secrets},
		{"secrets", "update", &caps.SecretsUpdate},
		{"secrets", "create", &caps.HelmWrite},
		{"nodes", "patch", &caps.NodeWrite},
	}

	var wg sync.WaitGroup
	wg.Add(len(checks))

	for _, check := range checks {
		go func(c capCheck) {
			defer wg.Done()
			allowed, _ := canIAs(checkCtx, k8sClient, username, groups, "", "", c.resource, c.verb)
			if allowed {
				*c.result = true
				return
			}
			// Try namespace-scoped fallback
			if fallbackNs := GetEffectiveNamespace(); fallbackNs != "" {
				allowed, _ = canIAs(checkCtx, k8sClient, username, groups, fallbackNs, "", c.resource, c.verb)
				if allowed {
					*c.result = true
				}
			}
		}(check)
	}

	wg.Wait()

	if ForceDisableHelmWrite {
		caps.HelmWrite = false
	}

	// Cache result
	userCapabilitiesCache.Store(username, &userCapEntry{
		caps:      caps,
		expiresAt: time.Now().Add(userCapabilitiesTTL),
	})

	return caps, nil
}

// canIAs checks if a specific user can perform an action using SubjectAccessReview.
// Unlike canI which uses SelfSubjectAccessReview (checks the ServiceAccount),
// this checks on behalf of a specific user.
func canIAs(ctx context.Context, client *kubernetes.Clientset, username string, groups []string, namespace, group, resource, verb string) (bool, bool) {
	if ctx.Err() != nil {
		return false, true
	}
	if client == nil {
		return false, true
	}

	review := &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			User:   username,
			Groups: groups,
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: namespace,
				Group:     group,
				Verb:      verb,
				Resource:  resource,
			},
		},
	}

	result, err := client.AuthorizationV1().SubjectAccessReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		if ctx.Err() == nil {
			log.Printf("Warning: SubjectAccessReview failed for user=%s %s %s: %v", SanitizeForLog(username), SanitizeForLog(verb), SanitizeForLog(resource), err)
		}
		return false, true
	}

	return result.Status.Allowed, false
}

// InvalidateUserCapabilitiesCache clears all per-user capability caches
func InvalidateUserCapabilitiesCache() {
	userCapabilitiesCache.Range(func(key, _ any) bool {
		userCapabilitiesCache.Delete(key)
		return true
	})
}

var (
	cachedPermResult      *PermissionCheckResult
	resourcePermsMu       sync.RWMutex
	resourcePermsExpiry   time.Time
	resourcePermsTTL      = 60 * time.Second
	resourcePermsErrorTTL = 5 * time.Second // Short TTL when API errors caused fail-closed results
)

// resourceProbe describes one typed-resource probe target. The probe issues
// `list?limit=1` against this resource (cluster-wide first, then namespace-scoped
// fallback for non-cluster-scoped kinds when a fallback namespace is set), and
// the result drives whether an informer is created and at what scope.
type resourceProbe struct {
	key         string                      // ResourceType key (k8score.Pods etc.)
	gvr         schema.GroupVersionResource // For dynamic-client probe
	clusterOnly bool                        // true: cannot be namespace-scoped (nodes, namespaces, PV, storageclasses)
	field       *bool                       // Pointer into the boolean view on ResourcePermissions
}

// resourceProbeTargets returns the typed informer kinds we probe access for.
// Includes Gateway / HTTPRoute even though they live in the dynamic cache —
// the boolean lives in ResourcePermissions and is consumed by the UI snapshot.
func resourceProbeTargets(perms *ResourcePermissions) []resourceProbe {
	return []resourceProbe{
		{k8score.Pods, schema.GroupVersionResource{Version: "v1", Resource: "pods"}, false, &perms.Pods},
		{k8score.Services, schema.GroupVersionResource{Version: "v1", Resource: "services"}, false, &perms.Services},
		{k8score.ConfigMaps, schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, false, &perms.ConfigMaps},
		{k8score.Secrets, schema.GroupVersionResource{Version: "v1", Resource: "secrets"}, false, &perms.Secrets},
		{k8score.Events, schema.GroupVersionResource{Version: "v1", Resource: "events"}, false, &perms.Events},
		{k8score.PersistentVolumeClaims, schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}, false, &perms.PersistentVolumeClaims},
		{k8score.ServiceAccounts, schema.GroupVersionResource{Version: "v1", Resource: "serviceaccounts"}, false, &perms.ServiceAccounts},
		{k8score.LimitRanges, schema.GroupVersionResource{Version: "v1", Resource: "limitranges"}, false, &perms.LimitRanges},
		{k8score.Nodes, schema.GroupVersionResource{Version: "v1", Resource: "nodes"}, true, &perms.Nodes},
		{k8score.Namespaces, schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}, true, &perms.Namespaces},
		{k8score.PersistentVolumes, schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumes"}, true, &perms.PersistentVolumes},
		{k8score.Deployments, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, false, &perms.Deployments},
		{k8score.DaemonSets, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, false, &perms.DaemonSets},
		{k8score.StatefulSets, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, false, &perms.StatefulSets},
		{k8score.ReplicaSets, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}, false, &perms.ReplicaSets},
		{k8score.Ingresses, schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}, false, &perms.Ingresses},
		{k8score.NetworkPolicies, schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}, false, &perms.NetworkPolicies},
		{k8score.Jobs, schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}, false, &perms.Jobs},
		{k8score.CronJobs, schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}, false, &perms.CronJobs},
		{k8score.HorizontalPodAutoscalers, schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}, false, &perms.HorizontalPodAutoscalers},
		{k8score.StorageClasses, schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"}, true, &perms.StorageClasses},
		{k8score.PodDisruptionBudgets, schema.GroupVersionResource{Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"}, false, &perms.PodDisruptionBudgets},
		{k8score.Roles, schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"}, false, &perms.Roles},
		{k8score.ClusterRoles, schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"}, true, &perms.ClusterRoles},
		{k8score.RoleBindings, schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"}, false, &perms.RoleBindings},
		{k8score.ClusterRoleBindings, schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"}, true, &perms.ClusterRoleBindings},
		// Gateway/HTTPRoute live in the dynamic cache, but the bool surfaces in the UI.
		{"gateways", schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}, false, &perms.Gateways},
		{"httproutes", schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}, false, &perms.HTTPRoutes},
	}
}

// SanitizeForLog strips CR/LF from a string before it's written to a log.
// Use this for any value that originates from user-controlled input
// (HTTP request bodies, kubeconfig fields edited by the user, etc.) —
// without it, an attacker-controlled string containing newlines could
// inject forged log entries. CodeQL's `Log entries created from user
// input` rule fires on tainted strings even when wrapped in %q because
// the taint analyzer doesn't model fmt's escaping behavior; an explicit
// strings.ReplaceAll terminates the taint flow.
func SanitizeForLog(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// probeListAccessWith attempts a list?limit=1 against the GVR using the
// given dynamic client. Returns:
//   - allowed=true: list succeeded — informer can run.
//   - allowed=false, forbidden=true: explicit 403/401 — gate the informer.
//   - allowed=true, transient!=nil: non-auth error (network, 503, NotFound for
//     missing CRD, etc.). Treated as "allow optimistically" so a transient API
//     hiccup doesn't permanently disable the resource for the session — the
//     informer's reflector will retry. Same convention as the dynamic cache
//     probe in pkg/k8score/dynamic_cache.go.
//
// Exposed (lowercase but called from tests in the same package) so tests can
// drive it with a fake dynamic.Interface.
func probeListAccessWith(ctx context.Context, dyn dynamic.Interface, gvr schema.GroupVersionResource, namespace string) (allowed bool, forbidden bool, transient error) {
	if dyn == nil {
		return false, false, nil
	}
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	opts := metav1.ListOptions{Limit: 1}
	var err error
	if namespace != "" {
		_, err = dyn.Resource(gvr).Namespace(namespace).List(probeCtx, opts)
	} else {
		_, err = dyn.Resource(gvr).List(probeCtx, opts)
	}
	if err == nil {
		return true, false, nil
	}
	if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) {
		return false, true, nil
	}
	return true, false, err
}

// CheckResourcePermissions probes list access for every typed resource and
// returns per-kind scope plus a uniform projection. Results are cached for
// 60s (5s on transient errors).
//
// Per-kind probe behavior:
//   - Cluster-wide list?limit=1 first.
//   - If 403/401 and the kind is namespaceable AND a fallback namespace is set,
//     retry scoped to that namespace.
//   - Anything still 403/401 → kind is denied.
//   - Anything that returns a non-auth error (transient, NotFound for a
//     missing CRD) → optimistically allowed cluster-wide.
//
// This is authoritative because it IS the operation the informer will perform.
// SelfSubjectAccessReview is one indirection too many — it can disagree with
// reality on clusters using webhook authorizers (e.g. GKE IAM).
func CheckResourcePermissions(ctx context.Context) *PermissionCheckResult {
	resourcePermsMu.RLock()
	if cachedPermResult != nil && time.Now().Before(resourcePermsExpiry) {
		// Deep-copy so callers can't mutate the cached result.
		permsCopy := *cachedPermResult.Perms
		scopesCopy := make(map[string]k8score.ResourceScope, len(cachedPermResult.Scopes))
		for k, v := range cachedPermResult.Scopes {
			scopesCopy[k] = v
		}
		result := &PermissionCheckResult{
			Perms:           &permsCopy,
			NamespaceScoped: cachedPermResult.NamespaceScoped,
			Namespace:       cachedPermResult.Namespace,
			Scopes:          scopesCopy,
		}
		resourcePermsMu.RUnlock()
		return result
	}
	resourcePermsMu.RUnlock()

	// Compute probes WITHOUT holding the write lock — concurrent callers
	// may race but redundant probes are harmless. Holding the lock during
	// network calls would block InvalidateResourcePermissionsCache() during
	// context switch.

	if GetClient() == nil || GetDynamicClient() == nil {
		log.Printf("Warning: K8s client not initialized, returning no resource permissions")
		return &PermissionCheckResult{Perms: &ResourcePermissions{}, Scopes: map[string]k8score.ResourceScope{}}
	}

	// scopeNs comes from the kubeconfig context namespace or --namespace
	// flag — a fallback used when cluster-wide access is denied. The cache
	// boots cluster-wide and per-user view filtering happens at the HTTP
	// layer (see internal/server/namespace_scope.go), so the probe never
	// pins informers to a single namespace on behalf of one user.
	scopeNs := GetEffectiveNamespace()

	result, hadErrors := probeResourceAccess(ctx, GetDynamicClient(), scopeNs, false)

	resourcePermsMu.Lock()
	cachedPermResult = result
	ttl := resourcePermsTTL
	if hadErrors {
		ttl = resourcePermsErrorTTL
		log.Printf("Warning: resource access probes had API errors, using short cache TTL (%v)", ttl)
	}
	resourcePermsExpiry = time.Now().Add(ttl)
	resourcePermsMu.Unlock()

	return result
}

// probeResourceAccess is the testable inner of CheckResourcePermissions.
// It does the actual probing with the supplied dynamic client and namespace,
// with no caching and no global state. The returned bool is true when at
// least one probe hit a non-auth (transient) error — caller uses this to
// shorten the cache TTL so the next attempt re-probes.
//
// scopeNs and forceNamespace together describe the namespace's role:
//   - forceNamespace=false: scopeNs is a kubeconfig fallback only. Probe
//     cluster-wide first; on 403 retry namespace-scoped against scopeNs.
//     This is the only path used by production code — the cache always boots
//     cluster-wide and per-user view filtering happens at the HTTP layer
//     (see internal/server/namespace_scope.go).
//   - forceNamespace=true: probe namespaced kinds ONLY in scopeNs. Reserved
//     for tests / a hypothetical future per-cache pin; not reachable from
//     CheckResourcePermissions today. Cluster-only kinds (nodes, namespaces,
//     PV, storageclasses, ingressclasses) are still probed cluster-wide
//     since they have no namespace dimension to pin to.
func probeResourceAccess(ctx context.Context, dyn dynamic.Interface, scopeNs string, forceNamespace bool) (*PermissionCheckResult, bool) {
	perms := &ResourcePermissions{}
	probes := resourceProbeTargets(perms)

	type probeOutcome struct {
		scope k8score.ResourceScope
	}
	outcomes := make([]probeOutcome, len(probes))

	logTiming("   [perms] Probing list access for %d typed resources (scopeNs=%q forced=%v)", len(probes), scopeNs, forceNamespace)
	probeStart := time.Now()
	var wg sync.WaitGroup
	var hadErrors atomic.Bool
	wg.Add(len(probes))

	for i, p := range probes {
		go func(i int, p resourceProbe) {
			defer wg.Done()

			if forceNamespace {
				// Cluster-only kinds (nodes, namespaces, PV…) have no namespace
				// dimension — pin them cluster-wide regardless of the picked
				// namespace, so a cluster-admin who scoped to a namespace still
				// sees Node counts, Namespace lists, and node metrics.
				if p.clusterOnly {
					allowed, _, transient := probeListAccessWith(ctx, dyn, p.gvr, "")
					if transient != nil {
						hadErrors.Store(true)
					}
					if allowed {
						outcomes[i] = probeOutcome{scope: k8score.ResourceScope{Enabled: true, Namespace: ""}}
					}
					return
				}
				if scopeNs == "" {
					return
				}
				nsAllowed, _, nsTransient := probeListAccessWith(ctx, dyn, p.gvr, scopeNs)
				if nsTransient != nil {
					hadErrors.Store(true)
				}
				if nsAllowed {
					outcomes[i] = probeOutcome{scope: k8score.ResourceScope{Enabled: true, Namespace: scopeNs}}
				}
				return
			}

			allowed, forbidden, transient := probeListAccessWith(ctx, dyn, p.gvr, "")
			if transient != nil {
				hadErrors.Store(true)
			}
			if allowed {
				outcomes[i] = probeOutcome{scope: k8score.ResourceScope{Enabled: true, Namespace: ""}}
				return
			}
			// Cluster-wide denied. Cluster-scoped kinds have no fallback.
			if !forbidden || p.clusterOnly || scopeNs == "" {
				return
			}
			nsAllowed, _, nsTransient := probeListAccessWith(ctx, dyn, p.gvr, scopeNs)
			if nsTransient != nil {
				hadErrors.Store(true)
			}
			if nsAllowed {
				outcomes[i] = probeOutcome{scope: k8score.ResourceScope{Enabled: true, Namespace: scopeNs}}
			}
		}(i, p)
	}

	wg.Wait()
	logTiming("    Probe phase (%d resources): %v", len(probes), time.Since(probeStart))

	if ctx.Err() != nil {
		logTiming("   [perms] Bailing after probes: context canceled")
		return &PermissionCheckResult{Perms: perms, Scopes: map[string]k8score.ResourceScope{}}, true
	}

	// Apply outcomes to perms (boolean projection) and build the scope map.
	scopes := make(map[string]k8score.ResourceScope, len(probes))
	namespaceScoped := false
	var (
		restricted   []string
		nsScopedKeys []string
	)
	for i, p := range probes {
		r := outcomes[i]
		scopes[p.key] = r.scope
		if r.scope.Enabled {
			*p.field = true
			if r.scope.Namespace != "" {
				namespaceScoped = true
				nsScopedKeys = append(nsScopedKeys, p.key)
			}
		} else {
			restricted = append(restricted, p.key)
		}
	}

	// scopeNs comes from operator-controlled config (kubeconfig context
	// namespace or --namespace flag), but strip CR/LF defensively so log
	// scrapers can't be tricked by a malicious kubeconfig. CodeQL's taint
	// analysis doesn't model %q escaping, so be explicit.
	logSafeNs := SanitizeForLog(scopeNs)
	if len(restricted) > 0 {
		sort.Strings(restricted)
		if namespaceScoped {
			sort.Strings(nsScopedKeys)
			log.Printf("RBAC: mixed scope (namespace=%q; ns-scoped: %s); denied: %s",
				logSafeNs, strings.Join(nsScopedKeys, ", "), strings.Join(restricted, ", "))
		} else {
			log.Printf("RBAC: restricted resources (no list permission): %s", strings.Join(restricted, ", "))
		}
	} else if namespaceScoped {
		sort.Strings(nsScopedKeys)
		log.Printf("RBAC: mixed scope (namespace=%q; ns-scoped: %s); all kinds accessible",
			logSafeNs, strings.Join(nsScopedKeys, ", "))
	}

	// In forced-namespace mode the user's intent is to be ns-scoped — even
	// if every typed probe failed (e.g. they picked a namespace they have
	// no access to). Force NamespaceScoped=true so the dynamic cache scopes
	// CRD informers to the same namespace and doesn't silently fall through
	// to cluster-wide watches.
	if forceNamespace && scopeNs != "" {
		namespaceScoped = true
	}

	return &PermissionCheckResult{
		Perms:           perms,
		NamespaceScoped: namespaceScoped,
		Namespace:       scopeNs,
		Scopes:          scopes,
	}, hadErrors.Load()
}

// GetCachedPermissionResult returns the cached permission check result, if
// available. Returns a deep copy so callers can mutate Perms or Scopes
// without corrupting the cache (mirrors the cache-hit path in
// CheckResourcePermissions).
func GetCachedPermissionResult() *PermissionCheckResult {
	resourcePermsMu.RLock()
	defer resourcePermsMu.RUnlock()
	if cachedPermResult == nil {
		return nil
	}
	permsCopy := *cachedPermResult.Perms
	scopesCopy := make(map[string]k8score.ResourceScope, len(cachedPermResult.Scopes))
	for k, v := range cachedPermResult.Scopes {
		scopesCopy[k] = v
	}
	return &PermissionCheckResult{
		Perms:           &permsCopy,
		NamespaceScoped: cachedPermResult.NamespaceScoped,
		Namespace:       cachedPermResult.Namespace,
		Scopes:          scopesCopy,
	}
}

// InvalidateResourcePermissionsCache forces the next CheckResourcePermissions call to refresh
func InvalidateResourcePermissionsCache() {
	resourcePermsMu.Lock()
	defer resourcePermsMu.Unlock()
	cachedPermResult = nil
}
