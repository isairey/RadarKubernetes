import { describe, it, expect } from 'vitest'
import { parseContextName } from './context-name'

describe('parseContextName', () => {
  describe('GKE', () => {
    it('parses zonal cluster context', () => {
      const r = parseContextName('gke_my-project_us-east1-b_prod-cluster')
      expect(r.provider).toBe('GKE')
      expect(r.account).toBe('my-project')
      expect(r.region).toBe('us-east1-b')
      expect(r.clusterName).toBe('prod-cluster')
    })

    it('parses regional cluster context', () => {
      const r = parseContextName('gke_my-project_us-central1_my-cluster')
      expect(r.provider).toBe('GKE')
      expect(r.region).toBe('us-central1')
      expect(r.clusterName).toBe('my-cluster')
    })

    it('handles cluster names with underscores in the suffix', () => {
      const r = parseContextName('gke_proj_us-east1-b_my_funky_cluster')
      expect(r.provider).toBe('GKE')
      expect(r.clusterName).toBe('my_funky_cluster')
    })

    it('does NOT match arbitrary 3-underscore strings (regression)', () => {
      // The pre-fix regex `^gke_([^_]+)_([^_]+)_(.+)$` matched anything
      // gke-prefixed with three underscores. Tightened: zone must contain
      // a digit (every real GCP zone does).
      const r = parseContextName('gke_a_b_c')
      expect(r.provider).toBeNull()
    })

    it('rejects non-GKE-zone middle segment', () => {
      const r = parseContextName('gke_proj_notazone_cluster')
      expect(r.provider).toBeNull()
      expect(r.clusterName).toBe('gke_proj_notazone_cluster')
    })
  })

  describe('EKS ARN', () => {
    it('parses standard EKS ARN', () => {
      const r = parseContextName('arn:aws:eks:us-east-1:123456789012:cluster/my-prod')
      expect(r.provider).toBe('EKS')
      expect(r.account).toBe('123456789012')
      expect(r.region).toBe('us-east-1')
      expect(r.clusterName).toBe('my-prod')
    })

    it('handles cluster names with hyphens', () => {
      const r = parseContextName('arn:aws:eks:eu-central-1:982081053473:cluster/us-east-1-nonprod')
      expect(r.clusterName).toBe('us-east-1-nonprod')
    })
  })

  describe('eksctl format', () => {
    it('parses user@cluster.region.eksctl.io', () => {
      const r = parseContextName('admin@my-cluster.us-east-1.eksctl.io')
      expect(r.provider).toBe('EKS')
      expect(r.account).toBe('eksctl')
      expect(r.region).toBe('us-east-1')
      expect(r.clusterName).toBe('my-cluster')
    })
  })

  describe('AKS', () => {
    it('parses clusterUser_<rg>_<name>', () => {
      const r = parseContextName('clusterUser_my-rg_my-cluster')
      expect(r.provider).toBe('AKS')
      expect(r.account).toBe('my-rg')
      expect(r.clusterName).toBe('my-cluster')
    })

    it('parses clusterAdmin_<rg>_<name>', () => {
      const r = parseContextName('clusterAdmin_prod-rg_prod-cluster')
      expect(r.provider).toBe('AKS')
      expect(r.account).toBe('prod-rg')
      expect(r.clusterName).toBe('prod-cluster')
    })

    it('does NOT mis-tag user-named clusters that happen to contain "aks"', () => {
      // Regression: pre-fix used substring match for "aks", so these
      // false-positive matched.
      expect(parseContextName('tracks-prod').provider).toBeNull()
      expect(parseContextName('my-aks-experiment').provider).toBeNull()
      expect(parseContextName('aks').provider).toBeNull()
    })
  })

  describe('unknown', () => {
    it('returns user-named cluster unchanged', () => {
      const r = parseContextName('my-staging-cluster')
      expect(r.provider).toBeNull()
      expect(r.clusterName).toBe('my-staging-cluster')
      expect(r.raw).toBe('my-staging-cluster')
    })

    it('returns empty-shape unknown for unparseable input', () => {
      const r = parseContextName('something-else-entirely')
      expect(r.provider).toBeNull()
      expect(r.account).toBeNull()
      expect(r.region).toBeNull()
    })
  })
})
