import { describe, it, expect } from 'vitest'
import { pluralize } from '../../utils/pluralize'
import type { SelectedKindInfo } from './ResourcesSidebar'

// Mirrors the empty-state hint rule inside ResourcesView (kept in
// sync by reuse) — pure so we can pin the lookup + render contract
// without rendering the React component. See
// `packages/k8s-ui/src/components/resources/ResourcesView.tsx`
// search for "the count is unfiltered".
function sidebarHint(
  counts: Record<string, number>,
  selectedKind: SelectedKindInfo,
  searchTerm: string,
): string | null {
  if (!searchTerm) return null
  const key = selectedKind.group ? `${selectedKind.group}/${selectedKind.kind}` : selectedKind.kind
  const totalForKind = counts[key] ?? 0
  if (totalForKind === 0) return null
  return `The sidebar shows ${pluralize(totalForKind, selectedKind.kind)} in the cluster — the count is unfiltered.`
}

describe('ResourcesView empty-search sidebar hint (SKY-828 bug 46)', () => {
  // The bug: sidebar count badge stays unfiltered and shows the
  // cluster-wide total per kind. When a search returns zero rows
  // (e.g. "5 Pods" in sidebar, "No Pods found" in pane) the user
  // reads the badge as a lie. The empty-state appends a sentence
  // making the unfiltered nature explicit. The rule must:
  //   1. Suppress entirely when there's no active search.
  //   2. Suppress when the cluster total for the kind is 0
  //      (no badge to explain → no hint).
  //   3. Look up the count under `${group}/${kind}` for grouped
  //      kinds, plain `kind` for core kinds (matches sidebar key).
  //   4. Pluralize via pluralize() so "Ingress" → "Ingresses",
  //      "NetworkPolicy" → "NetworkPolicies" (regression of the
  //      naive `${kind}s` Cursor Bugbot caught in 80eb64b).
  //   5. Singular for n===1: "1 Pod", not "1 Pods".

  const pod: SelectedKindInfo = { name: 'pods', kind: 'Pod', group: '' }

  it('returns null when there is no active search', () => {
    expect(sidebarHint({ Pod: 232 }, pod, '')).toBeNull()
  })

  it('returns null when the cluster total for the kind is 0', () => {
    expect(sidebarHint({ Pod: 0 }, pod, 'xyz')).toBeNull()
  })

  it('returns null when the kind is missing from counts', () => {
    expect(sidebarHint({}, pod, 'xyz')).toBeNull()
  })

  it('formats the sentence with the cluster total and pluralized kind', () => {
    expect(sidebarHint({ Pod: 232 }, pod, 'xyz')).toBe(
      'The sidebar shows 232 Pods in the cluster — the count is unfiltered.',
    )
  })

  it('uses singular noun when total is 1', () => {
    expect(sidebarHint({ Pod: 1 }, pod, 'xyz')).toBe(
      'The sidebar shows 1 Pod in the cluster — the count is unfiltered.',
    )
  })

  it('looks up grouped kinds under `${group}/${kind}`', () => {
    const ar: SelectedKindInfo = { name: 'rollouts', kind: 'Rollout', group: 'argoproj.io' }
    const counts = { Rollout: 999, 'argoproj.io/Rollout': 4 }
    expect(sidebarHint(counts, ar, 'xyz')).toBe(
      'The sidebar shows 4 Rollouts in the cluster — the count is unfiltered.',
    )
  })

  it('pluralizes Ingress correctly (regression: not "Ingresss")', () => {
    const ing: SelectedKindInfo = { name: 'ingresses', kind: 'Ingress', group: '' }
    expect(sidebarHint({ Ingress: 3 }, ing, 'xyz')).toBe(
      'The sidebar shows 3 Ingresses in the cluster — the count is unfiltered.',
    )
  })

  it('pluralizes NetworkPolicy correctly (regression: not "NetworkPolicys")', () => {
    const np: SelectedKindInfo = { name: 'networkpolicies', kind: 'NetworkPolicy', group: 'networking.k8s.io' }
    expect(sidebarHint({ 'networking.k8s.io/NetworkPolicy': 2 }, np, 'xyz')).toBe(
      'The sidebar shows 2 NetworkPolicies in the cluster — the count is unfiltered.',
    )
  })
})
