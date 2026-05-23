package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skyhook-io/radar/internal/k8s"
	pkgauth "github.com/skyhook-io/radar/pkg/auth"
)

// newTestServer constructs a Server with just the state needed by the
// namespace-pick helpers. Avoids the full New() path so we can drive the
// helpers directly without spinning up auth providers or a router.
//
// Only restores the context name on cleanup — does NOT call ResetTestState
// (which would nuke the connection state TestMain established).
func newTestServer(t *testing.T) *Server {
	t.Helper()
	prev := k8s.SetTestContextName("test-ctx")
	t.Cleanup(func() { k8s.SetTestContextName(prev) })
	return &Server{}
}

func reqAs(username string) *http.Request {
	r := httptest.NewRequest("GET", "/api/cluster/namespace", nil)
	if username != "" {
		ctx := pkgauth.ContextWithUser(r.Context(), &pkgauth.User{Username: username})
		r = r.WithContext(ctx)
	}
	return r
}

func TestNsPreferenceKey_PerUserIsolation(t *testing.T) {
	// Different users must produce distinct keys. Without the username,
	// one user's pick would shadow another's.
	if nsPreferenceKey("alice", "ctx") == nsPreferenceKey("bob", "ctx") {
		t.Error("alice and bob produced the same nsPreferenceKey")
	}
	// Same user, same context — keys must match.
	if nsPreferenceKey("alice", "ctx") != nsPreferenceKey("alice", "ctx") {
		t.Error("nsPreferenceKey is not deterministic")
	}
	// Empty username (no-auth) collapses to a per-context key.
	if nsPreferenceKey("", "ctx-a") == nsPreferenceKey("", "ctx-b") {
		t.Error("no-auth keys for different contexts should differ")
	}
	// Substring confusion: alice/foo must not collide with alic/efoo etc.
	// The \x00 separator makes this safe — verify by counterexample.
	if nsPreferenceKey("alice", "foo") == nsPreferenceKey("ali", "cefoo") {
		t.Error("nsPreferenceKey separator is ambiguous")
	}
}

func TestSetAndGetActiveNamespaceForUser_PerUser(t *testing.T) {
	s := newTestServer(t)

	// Alice picks alpha; Bob picks beta. Each must read back their own pick.
	s.setActiveNamespaceForUser(reqAs("alice"), "alpha")
	s.setActiveNamespaceForUser(reqAs("bob"), "beta")

	if got := s.getActiveNamespaceForUser(reqAs("alice")); got != "alpha" {
		t.Errorf("alice: got %q, want alpha", got)
	}
	if got := s.getActiveNamespaceForUser(reqAs("bob")); got != "beta" {
		t.Errorf("bob: got %q, want beta", got)
	}

	// A third user with no pick gets the empty default.
	if got := s.getActiveNamespaceForUser(reqAs("carol")); got != "" {
		t.Errorf("carol: expected empty pick, got %q", got)
	}
}

func TestSetActiveNamespaceForUser_EmptyClears(t *testing.T) {
	s := newTestServer(t)

	s.setActiveNamespaceForUser(reqAs("alice"), "alpha")
	s.setActiveNamespaceForUser(reqAs("alice"), "") // clear

	if got := s.getActiveNamespaceForUser(reqAs("alice")); got != "" {
		t.Errorf("expected empty after clear, got %q", got)
	}
}

func TestSetActiveNamespaceForUser_NoAuth(t *testing.T) {
	s := newTestServer(t)

	// Auth disabled — empty username path. The key is still per-context.
	s.setActiveNamespaceForUser(reqAs(""), "alpha")
	if got := s.getActiveNamespaceForUser(reqAs("")); got != "alpha" {
		t.Errorf("no-auth: got %q, want alpha", got)
	}
}

func TestSetActiveNamespaceForUser_NoContext(t *testing.T) {
	// When no kubeconfig context is set (e.g. before initial connection),
	// set/get must be no-ops — there's no cluster to scope to.
	prev := k8s.SetTestContextName("")
	t.Cleanup(func() { k8s.SetTestContextName(prev) })
	s := &Server{}

	s.setActiveNamespaceForUser(reqAs("alice"), "alpha")
	if got := s.getActiveNamespaceForUser(reqAs("alice")); got != "" {
		t.Errorf("expected empty without context, got %q", got)
	}
}

func TestClearAllNamespacePreferences(t *testing.T) {
	s := newTestServer(t)

	s.setActiveNamespaceForUser(reqAs("alice"), "alpha")
	s.setActiveNamespaceForUser(reqAs("bob"), "beta")
	s.setActiveNamespaceForUser(reqAs(""), "gamma")

	s.clearAllNamespacePreferences()

	for _, user := range []string{"alice", "bob", ""} {
		if got := s.getActiveNamespaceForUser(reqAs(user)); got != "" {
			t.Errorf("user=%q: expected cleared, got %q", user, got)
		}
	}
}

func TestFinalizePostContextSwitch_ClearsBothCaches(t *testing.T) {
	// Pin the load-bearing claim from the comment on finalizePostContextSwitch:
	// it MUST clear permCache AND every user's namespace pick. A regression
	// that drops either side leaves stale state attached to the new cluster.
	s := newTestServer(t)
	s.permCache = pkgauth.NewPermissionCache()
	s.permCache.Set("alice", &pkgauth.UserPermissions{AllowedNamespaces: []string{"alpha"}})
	s.setActiveNamespaceForUser(reqAs("alice"), "alpha")
	s.setActiveNamespaceForUser(reqAs("bob"), "beta")

	s.finalizePostContextSwitch()

	if got := s.permCache.Get("alice"); got != nil {
		t.Errorf("permCache.Get(alice) = %+v after finalize, want nil", got)
	}
	if got := s.getActiveNamespaceForUser(reqAs("alice")); got != "" {
		t.Errorf("alice ns pick survived: %q", got)
	}
	if got := s.getActiveNamespaceForUser(reqAs("bob")); got != "" {
		t.Errorf("bob ns pick survived: %q", got)
	}
}

func TestFinalizePostContextSwitch_NilPermCacheNoCrash(t *testing.T) {
	// finalizePostContextSwitch is called from CAPI connect / context switch
	// before s.permCache may have been initialized in some paths; guarding
	// nil is the contract.
	s := newTestServer(t)
	s.permCache = nil
	s.setActiveNamespaceForUser(reqAs("alice"), "alpha")

	s.finalizePostContextSwitch() // must not panic

	if got := s.getActiveNamespaceForUser(reqAs("alice")); got != "" {
		t.Errorf("ns pick survived nil-permCache finalize: %q", got)
	}
}

func TestClearAllNamespacePreferences_OnContextSwitch(t *testing.T) {
	// Picks made under context A must not survive a switch to context B —
	// they reference namespaces that don't exist on the new cluster.
	s := newTestServer(t)

	k8s.SetTestContextName("ctx-a")
	s.setActiveNamespaceForUser(reqAs("alice"), "alpha")

	// Switch context (callers do this via PerformContextSwitch which calls
	// clearAllNamespacePreferences before swapping context).
	s.clearAllNamespacePreferences()
	k8s.SetTestContextName("ctx-b")

	if got := s.getActiveNamespaceForUser(reqAs("alice")); got != "" {
		t.Errorf("pick survived context switch: got %q", got)
	}
}
