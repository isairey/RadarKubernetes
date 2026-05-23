package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/skyhook-io/radar/internal/k8s"
	pkgauth "github.com/skyhook-io/radar/pkg/auth"
)

// These tests verify that each MCP read handler integrates with the
// per-user filter helpers — i.e. results are narrowed to the calling
// user's allowed namespaces, cluster-only kinds are hidden from
// non-cluster-admins, and no-auth callers see everything in the cache.
//
// They use a real (fake-backed) ResourceCache rather than mocks so the
// data path matches production: cache holds objects across namespaces;
// filtering happens at handler-call time, not at cache-population time.

func setupFakeCacheForFilterTests(t *testing.T) {
	t.Helper()

	fakeClient := fake.NewClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "alpha"}, Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "beta"}, Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "gamma"}, Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive}},

		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "alpha-pod", Namespace: "alpha"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "beta-pod", Namespace: "beta"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "gamma-pod", Namespace: "gamma"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},

		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-2"}},
	)

	if err := k8s.InitTestResourceCache(fakeClient); err != nil {
		t.Fatalf("InitTestResourceCache: %v", err)
	}
	t.Cleanup(func() {
		k8s.ResetTestState()
		getPermCache().Invalidate()
	})

	// Mark connected so handlers don't bail early on requireConnected-style
	// checks. (The MCP handlers don't use requireConnected directly, but
	// downstream code paths do.)
	k8s.SetConnectionStatus(k8s.ConnectionStatus{State: k8s.StateConnected, Context: "fake-test"})
}

// withRestrictedUser primes the perm cache for a namespace-restricted user
// (allowed = exactly the namespaces passed) and returns a context with that
// user attached. Use nil/empty allowed for "denied to all" testing.
//
// The user starts with NO cluster-scoped read permissions cached. Call
// grantClusterRead in the test to seed specific allowed (group, resource)
// tuples — anything not seeded will trigger a live SAR which fails closed
// (no K8s client in the test harness).
func withRestrictedUser(t *testing.T, username string, allowed []string) context.Context {
	t.Helper()
	ctx := pkgauth.ContextWithUser(context.Background(), &pkgauth.User{Username: username, Groups: nil})
	getPermCache().Set(username, &pkgauth.UserPermissions{AllowedNamespaces: allowed})
	return ctx
}

// withClusterAdmin attaches a user whose perms have nil AllowedNamespaces
// (cluster-wide namespaced access). NOTE: this no longer implies cluster-
// scoped read access — those gates are per-(group, resource) SARs. Call
// grantClusterRead to seed specific cluster-scoped reads as authorized.
func withClusterAdmin(t *testing.T, username string) context.Context {
	t.Helper()
	ctx := pkgauth.ContextWithUser(context.Background(), &pkgauth.User{Username: username, Groups: nil})
	getPermCache().Set(username, &pkgauth.UserPermissions{AllowedNamespaces: nil})
	return ctx
}

// grantClusterRead seeds the per-kind SAR cache so canReadClusterScopedKind
// returns the desired result without making a live call. Each gvr is
// "<group>/<resource>" (group="" for core), e.g. "/nodes", "rbac.authorization.k8s.io/clusterroles".
// Sets both list and get verbs.
func grantClusterRead(t *testing.T, username string, gvrs ...string) {
	t.Helper()
	perms := getPermCache().Get(username)
	if perms == nil {
		t.Fatalf("user %q not in perm cache; call withRestrictedUser/withClusterAdmin first", username)
	}
	for _, gvr := range gvrs {
		group, resource, ok := strings.Cut(gvr, "/")
		if !ok {
			t.Fatalf("grantClusterRead: bad GVR %q (need group/resource)", gvr)
		}
		perms.SetCanI("list", group, resource, "", true)
		perms.SetCanI("get", group, resource, "", true)
	}
}

// denyClusterRead is the explicit-deny counterpart. Use to verify cluster-
// scoped reads are gated even when the cache contains the resource.
func denyClusterRead(t *testing.T, username string, gvrs ...string) {
	t.Helper()
	perms := getPermCache().Get(username)
	if perms == nil {
		t.Fatalf("user %q not in perm cache", username)
	}
	for _, gvr := range gvrs {
		group, resource, ok := strings.Cut(gvr, "/")
		if !ok {
			t.Fatalf("denyClusterRead: bad GVR %q", gvr)
		}
		perms.SetCanI("list", group, resource, "", false)
		perms.SetCanI("get", group, resource, "", false)
	}
}

// extractText pulls the JSON-encoded payload out of an MCP CallToolResult.
// Our handlers always return a single TextContent block (toJSONResult marshals
// the data and wraps it in TextContent.Text).
func extractText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil || len(result.Content) == 0 {
		t.Fatalf("empty CallToolResult")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

// containsName checks the JSON payload for an object with the given name.
// Strict-string matching is fine because pod/namespace names in our fixture
// are unique.
func containsName(payload, name string) bool {
	return strings.Contains(payload, `"name":"`+name+`"`)
}

func TestHandleListResources_RestrictedUser(t *testing.T) {
	setupFakeCacheForFilterTests(t)

	// Alice can see "alpha" only.
	ctx := withRestrictedUser(t, "alice", []string{"alpha"})

	// list pods (no namespace param) — should return alpha-pod only.
	result, _, err := handleListResources(ctx, nil, listResourcesInput{Kind: "pods"})
	if err != nil {
		t.Fatalf("handleListResources: %v", err)
	}
	body := extractText(t, result)
	if !containsName(body, "alpha-pod") {
		t.Errorf("expected alpha-pod in result; got: %s", body)
	}
	if containsName(body, "beta-pod") || containsName(body, "gamma-pod") {
		t.Errorf("restricted user leaked other-namespace pods: %s", body)
	}
}

func TestHandleListResources_DeniedNamespace(t *testing.T) {
	setupFakeCacheForFilterTests(t)
	ctx := withRestrictedUser(t, "alice", []string{"alpha"})

	// Alice asks for beta — empty result, no error.
	result, _, err := handleListResources(ctx, nil, listResourcesInput{Kind: "pods", Namespace: "beta"})
	if err != nil {
		t.Fatalf("handleListResources: %v", err)
	}
	body := extractText(t, result)
	if containsName(body, "beta-pod") {
		t.Errorf("denied namespace leaked: %s", body)
	}
}

func TestHandleListResources_ClusterOnlyKindBlockedForRestricted(t *testing.T) {
	setupFakeCacheForFilterTests(t)
	// Alice is namespace-restricted to alpha and lacks Node read RBAC.
	// Seed an explicit deny so the test doesn't make a live SAR.
	ctx := withRestrictedUser(t, "alice", []string{"alpha"})
	denyClusterRead(t, "alice", "/nodes")

	result, _, err := handleListResources(ctx, nil, listResourcesInput{Kind: "nodes"})
	if err != nil {
		t.Fatalf("handleListResources: %v", err)
	}
	body := extractText(t, result)
	if containsName(body, "node-1") || containsName(body, "node-2") {
		t.Errorf("restricted user saw cluster-only Node resources: %s", body)
	}
}

func TestHandleListResources_ClusterWidePodsButNoNodes(t *testing.T) {
	// Cluster-wide namespaced read access (AllowedNamespaces==nil) does NOT
	// imply cluster-scoped read access. Nodes still require a successful
	// per-kind SAR; deny it here and verify nothing leaks from the cache.
	setupFakeCacheForFilterTests(t)
	ctx := withClusterAdmin(t, "broad-pod-reader") // nil AllowedNamespaces
	denyClusterRead(t, "broad-pod-reader", "/nodes")

	result, _, err := handleListResources(ctx, nil, listResourcesInput{Kind: "nodes"})
	if err != nil {
		t.Fatalf("handleListResources: %v", err)
	}
	body := extractText(t, result)
	if containsName(body, "node-1") || containsName(body, "node-2") {
		t.Errorf("user with cluster-wide pods but no nodes saw nodes: %s", body)
	}
}

func TestHandleGetDashboard_ClusterWidePodsButNoClusterScopedReads(t *testing.T) {
	setupFakeCacheForFilterTests(t)
	ctx := withClusterAdmin(t, "dashboard-viewer")
	denyClusterRead(t, "dashboard-viewer", "/nodes", "/namespaces")

	result, _, err := handleGetDashboard(ctx, nil, dashboardInput{})
	if err != nil {
		t.Fatalf("handleGetDashboard: %v", err)
	}

	var body mcpDashboard
	if err := json.Unmarshal([]byte(extractText(t, result)), &body); err != nil {
		t.Fatalf("unmarshal dashboard: %v", err)
	}
	if got := body.ResourceCounts["pods"]; got != 3 {
		t.Fatalf("expected cluster-wide pod count to remain visible, got %d", got)
	}
	if got := body.ResourceCounts["nodes"]; got != 0 {
		t.Fatalf("node count leaked without node read RBAC: %d", got)
	}
	if got := body.ResourceCounts["namespaces"]; got != 0 {
		t.Fatalf("namespace count leaked without namespace read RBAC: %d", got)
	}
	if body.Nodes.Total != 0 || len(body.VersionSkew) != 0 || body.TopologyNodes != 0 {
		t.Fatalf("cluster-scoped dashboard fields leaked: nodes=%+v versionSkew=%v topologyNodes=%d", body.Nodes, body.VersionSkew, body.TopologyNodes)
	}
}

func TestHandleListResources_ClusterAdminSeesEverything(t *testing.T) {
	setupFakeCacheForFilterTests(t)
	ctx := withClusterAdmin(t, "admin")
	grantClusterRead(t, "admin", "/nodes") // explicitly granted via seeded SAR

	// Pods: cluster-wide namespaced reads.
	result, _, err := handleListResources(ctx, nil, listResourcesInput{Kind: "pods"})
	if err != nil {
		t.Fatalf("handleListResources: %v", err)
	}
	body := extractText(t, result)
	for _, want := range []string{"alpha-pod", "beta-pod", "gamma-pod"} {
		if !containsName(body, want) {
			t.Errorf("cluster-admin missing %s: %s", want, body)
		}
	}

	// Nodes: granted via explicit SAR seed.
	result, _, err = handleListResources(ctx, nil, listResourcesInput{Kind: "nodes"})
	if err != nil {
		t.Fatalf("handleListResources nodes: %v", err)
	}
	body = extractText(t, result)
	if !containsName(body, "node-1") {
		t.Errorf("cluster-admin missing node-1: %s", body)
	}
}

func TestHandleListResources_NoAuthPassthrough(t *testing.T) {
	setupFakeCacheForFilterTests(t)

	// No user on context — every call passes through (local-binary case).
	result, _, err := handleListResources(context.Background(), nil, listResourcesInput{Kind: "pods"})
	if err != nil {
		t.Fatalf("handleListResources: %v", err)
	}
	body := extractText(t, result)
	for _, want := range []string{"alpha-pod", "beta-pod", "gamma-pod"} {
		if !containsName(body, want) {
			t.Errorf("no-auth passthrough missing %s: %s", want, body)
		}
	}
}

func TestHandleGetResource_DeniedNamespace(t *testing.T) {
	setupFakeCacheForFilterTests(t)
	ctx := withRestrictedUser(t, "alice", []string{"alpha"})

	_, _, err := handleGetResource(ctx, nil, getResourceInput{Kind: "pods", Namespace: "beta", Name: "beta-pod"})
	if err == nil {
		t.Fatal("expected forbidden error for denied namespace, got nil")
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("expected 'forbidden' in error, got: %v", err)
	}
}

func TestHandleGetResource_ClusterOnlyForbiddenForRestricted(t *testing.T) {
	setupFakeCacheForFilterTests(t)
	ctx := withRestrictedUser(t, "alice", []string{"alpha"})
	denyClusterRead(t, "alice", "/nodes")

	// Cluster-scoped get (empty namespace) requires explicit cluster-scoped
	// read RBAC; this user has neither namespace access nor seeded SAR allow.
	_, _, err := handleGetResource(ctx, nil, getResourceInput{Kind: "nodes", Namespace: "", Name: "node-1"})
	if err == nil {
		t.Fatal("expected forbidden error for cluster-scoped read, got nil")
	}
}

func TestHandleListResources_NamespacesRequiresListNamespacesSAR(t *testing.T) {
	// list_resources(kind=namespaces) returns full Namespace objects.
	// AllowedNamespaces==nil (cluster-wide-pods sentinel) does NOT license
	// Namespace metadata reads. Pin the strict gate; the synthesized
	// list_namespaces tool covers the picker UX for restricted users.
	setupFakeCacheForFilterTests(t)
	ctx := withClusterAdmin(t, "alice")
	denyClusterRead(t, "alice", "/namespaces")

	result, _, err := handleListResources(ctx, nil, listResourcesInput{Kind: "namespaces"})
	if err != nil {
		t.Fatalf("handleListResources: %v", err)
	}
	body := extractText(t, result)
	if containsName(body, "alpha") || containsName(body, "beta") || containsName(body, "gamma") {
		t.Errorf("namespaces leaked without list-namespaces SAR: %s", body)
	}
}

func TestHandleGetResource_NamespacesRequiresGetNamespacesSAR(t *testing.T) {
	// get_resource(kind=namespaces, name=alpha) returns the full Namespace
	// object. Read access to resources IN alpha is not the same as
	// get-namespace-alpha (which needs ClusterRole on namespaces).
	setupFakeCacheForFilterTests(t)
	ctx := withRestrictedUser(t, "alice", []string{"alpha"})
	denyClusterRead(t, "alice", "/namespaces")

	_, _, err := handleGetResource(ctx, nil, getResourceInput{Kind: "namespaces", Name: "alpha"})
	if err == nil {
		t.Fatal("expected forbidden error without get-namespaces SAR, got nil")
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("expected 'forbidden' in error, got: %v", err)
	}
}

func TestHandleListNamespaces_FiltersToAllowed(t *testing.T) {
	setupFakeCacheForFilterTests(t)
	ctx := withRestrictedUser(t, "alice", []string{"alpha", "beta"})

	result, _, err := handleListNamespaces(ctx, nil, struct{}{})
	if err != nil {
		t.Fatalf("handleListNamespaces: %v", err)
	}
	body := extractText(t, result)
	if !containsName(body, "alpha") || !containsName(body, "beta") {
		t.Errorf("expected alpha and beta in namespace list: %s", body)
	}
	if containsName(body, "gamma") {
		t.Errorf("denied namespace gamma leaked: %s", body)
	}
}

func TestHandleListNamespaces_NoAuth(t *testing.T) {
	setupFakeCacheForFilterTests(t)

	result, _, err := handleListNamespaces(context.Background(), nil, struct{}{})
	if err != nil {
		t.Fatalf("handleListNamespaces: %v", err)
	}
	body := extractText(t, result)
	for _, want := range []string{"alpha", "beta", "gamma"} {
		if !containsName(body, want) {
			t.Errorf("no-auth missing %s: %s", want, body)
		}
	}
}

func TestHandleGetEvents_RestrictedAggregatesAllowed(t *testing.T) {
	setupFakeCacheForFilterTests(t)
	ctx := withRestrictedUser(t, "alice", []string{"alpha"})

	// No events in the fake cache, but the call should not error and should
	// not attempt to read beta/gamma. We're verifying the empty-result path
	// short-circuits cleanly.
	result, _, err := handleGetEvents(ctx, nil, eventsInput{Namespace: "beta"})
	if err != nil {
		t.Fatalf("handleGetEvents: %v", err)
	}
	body := extractText(t, result)
	if !strings.Contains(body, "[]") {
		t.Errorf("expected empty result for denied namespace, got: %s", body)
	}
}
