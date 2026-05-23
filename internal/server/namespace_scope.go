package server

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"

	"github.com/skyhook-io/radar/internal/auth"
	"github.com/skyhook-io/radar/internal/k8s"
	"github.com/skyhook-io/radar/internal/settings"
)

// NamespaceScopeMode is the closed enum the frontend's `mode` discriminator
// expects. Mirrors the TS union in web/src/api/client.ts; a typo on either
// side breaks the wire contract.
type NamespaceScopeMode string

const (
	NamespaceScopeClusterWide NamespaceScopeMode = "cluster-wide"
	NamespaceScopeNamespace   NamespaceScopeMode = "namespace"
	NamespaceScopeRestricted  NamespaceScopeMode = "restricted"
)

// NamespaceScopeResponse describes this user's namespace-pick state.
//
// The picker is a per-user view filter — it does NOT mutate the shared cache.
// Picking a namespace narrows what THIS user sees on subsequent reads to
// the intersection of (their pick) and (their RBAC-allowed namespaces).
//
//   - Active is the user's current pick ("" = "All namespaces", no narrowing).
//   - Mode is "cluster-wide" when no pick is set and the user can list
//     namespaces, "namespace" when a pick is in effect, or "restricted"
//     when the user has no cluster-wide list access and hasn't picked one.
//   - AccessibleNamespaces is the picker source — what the user can choose
//     from. Authoritative=false means it's a best-effort short list (the
//     user lacks list-namespace RBAC; other namespaces may exist).
type NamespaceScopeResponse struct {
	Active               string             `json:"active"`
	KubeconfigNamespace  string             `json:"kubeconfigNamespace"`
	Mode                 NamespaceScopeMode `json:"mode"`
	AccessibleNamespaces []string           `json:"accessibleNamespaces"`
	Authoritative        bool               `json:"authoritative"`
	CanClearNamespace    bool               `json:"canClearNamespace"`
}

// nsPreferenceKey builds the per-user, per-context key for nsPreferences.
// Empty username (auth disabled) collapses to a per-context key, matching
// the local single-user expectation.
func nsPreferenceKey(username, contextName string) string {
	return username + "\x00" + contextName
}

// getActiveNamespaceForUser returns this user's namespace pick for the
// current context. Empty string means "All namespaces."
func (s *Server) getActiveNamespaceForUser(r *http.Request) string {
	username := ""
	if u := auth.UserFromContext(r.Context()); u != nil {
		username = u.Username
	}
	ctxName := k8s.GetContextName()
	if ctxName == "" {
		return ""
	}
	v, ok := s.nsPreferences.Load(nsPreferenceKey(username, ctxName))
	if !ok {
		return ""
	}
	return v.(string)
}

// setActiveNamespaceForUser updates this user's pick for the current context.
// Pass "" to clear (back to "All namespaces").
func (s *Server) setActiveNamespaceForUser(r *http.Request, namespace string) {
	username := ""
	if u := auth.UserFromContext(r.Context()); u != nil {
		username = u.Username
	}
	ctxName := k8s.GetContextName()
	if ctxName == "" {
		return
	}
	key := nsPreferenceKey(username, ctxName)
	if namespace == "" {
		s.nsPreferences.Delete(key)
		return
	}
	s.nsPreferences.Store(key, namespace)
}

// clearAllNamespacePreferences drops every saved pick. Called on context
// switch — picks against the previous cluster's namespaces are meaningless.
func (s *Server) clearAllNamespacePreferences() {
	s.nsPreferences.Range(func(k, _ any) bool {
		s.nsPreferences.Delete(k)
		return true
	})
}

// finalizePostContextSwitch clears all per-user state that referenced the
// previous cluster. Order is load-bearing: callers MUST run this AFTER
// PerformContextSwitch, never before — running it first opens a window
// where an in-flight request repopulates permCache with the OLD cluster's
// SAR results, and those entries (TTL 2m) then authorize NEW cluster
// requests.
func (s *Server) finalizePostContextSwitch() {
	if s.permCache != nil {
		s.permCache.Invalidate()
	}
	k8s.InvalidateUserCapabilitiesCache()
	s.clearAllNamespacePreferences()
}

// loadSavedNamespacePreference seeds the per-user map from settings.json on
// first reach. Only relevant for the no-auth (local single-user) path —
// auth-enabled deploys don't persist picks across pod restarts.
func (s *Server) loadSavedNamespacePreference(r *http.Request) {
	if auth.UserFromContext(r.Context()) != nil {
		return // multi-user: no shared persisted pref
	}
	ctxName := k8s.GetContextName()
	if ctxName == "" {
		return
	}
	key := nsPreferenceKey("", ctxName)
	if _, ok := s.nsPreferences.Load(key); ok {
		return
	}
	saved := settings.Load()
	if saved.ActiveNamespaces == nil {
		return
	}
	if ns := saved.ActiveNamespaces[ctxName]; ns != "" {
		s.nsPreferences.Store(key, ns)
	}
}

func (s *Server) handleGetNamespaceScope(w http.ResponseWriter, r *http.Request) {
	if !s.requireConnected(w) {
		return
	}

	s.loadSavedNamespacePreference(r)
	active := s.getActiveNamespaceForUser(r)
	kubeNs := k8s.GetContextNamespace()

	// What the SA / kubeconfig identity sees — used as the input set for
	// per-user filtering below. authoritative=true means "we got a real
	// list from the apiserver"; false means "best-effort short list".
	saAccessible, authoritative := k8s.GetAccessibleNamespaces(r.Context())

	// Intersect with the calling user's RBAC-allowed namespaces. For
	// no-auth callers and cluster-admin users, this is a pass-through
	// (returns saAccessible unchanged). For namespace-restricted users,
	// it returns only the namespaces they can list. authoritative drops to
	// false in the restricted case — the picker UI shows the "limited
	// visibility" affordance accordingly.
	namespaces := saAccessible
	if filtered := s.getUserNamespaces(r, saAccessible); filtered != nil {
		namespaces = filtered
		// If the per-user filter shrank the set, the "authoritative" claim
		// no longer applies — we don't know whether namespaces beyond the
		// user's RBAC exist (yes, they do; but the user can't act on them).
		if len(filtered) < len(saAccessible) {
			authoritative = false
		}
	}

	// If the user picked a namespace they no longer have access to (RBAC
	// changed mid-session), drop the stale pick so the UI doesn't render
	// a phantom selection.
	if active != "" && !slices.Contains(namespaces, active) {
		s.setActiveNamespaceForUser(r, "")
		active = ""
	}

	mode := NamespaceScopeClusterWide
	switch {
	case active != "":
		mode = NamespaceScopeNamespace
	case !authoritative:
		mode = NamespaceScopeRestricted
	}

	// canClear reports whether widening back to "All namespaces" is allowed
	// — cluster-wide list access (authoritative) is sufficient; otherwise we
	// require a kubeconfig or --namespace fallback so the UI has something
	// to fall back to.
	canClear := authoritative || k8s.HasNamespaceFallback()

	s.writeJSON(w, NamespaceScopeResponse{
		Active:               active,
		KubeconfigNamespace:  kubeNs,
		Mode:                 mode,
		AccessibleNamespaces: namespaces,
		Authoritative:        authoritative,
		CanClearNamespace:    canClear,
	})
}

type setActiveNamespaceRequest struct {
	// Namespace to focus on. Empty string clears the pick (= "All namespaces"
	// up to the user's RBAC ceiling).
	Namespace string `json:"namespace"`
}

func (s *Server) handleSetActiveNamespace(w http.ResponseWriter, r *http.Request) {
	if !s.requireConnected(w) {
		return
	}

	var req setActiveNamespaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Verify the user actually has access to the namespace they're picking.
	// For namespace-restricted users, picking a namespace they can't see
	// would create a phantom selection that returns nothing — and would
	// be a quiet info-leak (server-side acknowledgement of a namespace's
	// existence). Use the per-user filtered set, not the SA's set.
	if req.Namespace != "" {
		filtered := s.getUserNamespaces(r, []string{req.Namespace})
		// filtered semantics: nil = no filter (auth off / cluster-admin),
		// empty = denied, populated = allowed.
		if filtered != nil && len(filtered) == 0 {
			s.writeError(w, http.StatusForbidden, "no access to namespace "+req.Namespace)
			return
		}
		// For cluster-admin (filtered == nil), still verify the namespace
		// exists from the SA's view — picking a typo'd namespace should fail.
		if filtered == nil {
			accessible, _ := k8s.GetAccessibleNamespaces(r.Context())
			if !slices.Contains(accessible, req.Namespace) {
				s.writeError(w, http.StatusForbidden, "no access to namespace "+req.Namespace)
				return
			}
		}
	}

	s.setActiveNamespaceForUser(r, req.Namespace)

	// Persist the no-auth (single-user) pick across restarts. Auth-enabled
	// deploys skip persistence — it'd require user-keyed storage we don't
	// have. The in-memory pick already took effect, so a persistence failure
	// is non-fatal — we log and continue.
	if auth.UserFromContext(r.Context()) == nil {
		ctxName := k8s.GetContextName()
		if ctxName != "" {
			if _, err := settings.Update(func(st *settings.Settings) {
				if st.ActiveNamespaces == nil {
					st.ActiveNamespaces = map[string]string{}
				}
				if req.Namespace == "" {
					delete(st.ActiveNamespaces, ctxName)
				} else {
					st.ActiveNamespaces[ctxName] = req.Namespace
				}
			}); err != nil {
				log.Printf("[namespace] failed to persist namespace pick for context %q: %v", ctxName, err)
			}
		}
	}

	// Return the fresh scope state so the UI can update without a follow-up GET.
	s.handleGetNamespaceScope(w, r)
}
