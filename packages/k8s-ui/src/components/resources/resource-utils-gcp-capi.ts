// GCP CAPI Infrastructure Provider utility functions

import type { StatusBadge } from './resource-utils'
import { getCAPIReadyStatus } from './resource-utils-capi'

// ============================================================================
// GCPManagedControlPlane
// ============================================================================

export function getGCPMCPStatus(resource: any): StatusBadge {
  return getCAPIReadyStatus(resource)
}

export function getGCPMCPClusterName(resource: any): string {
  return resource.spec?.clusterName || resource.metadata?.name || '-'
}

export function getGCPMCPProject(resource: any): string {
  return resource.spec?.project || '-'
}

export function getGCPMCPLocation(resource: any): string {
  return resource.spec?.location || '-'
}

export function getGCPMCPVersion(resource: any): string {
  return resource.status?.version || resource.spec?.version || '-'
}

export function getGCPMCPReleaseChannel(resource: any): string {
  return resource.spec?.releaseChannel || '-'
}

export function getGCPMCPAutopilot(resource: any): boolean {
  return !!resource.spec?.enableAutopilot
}

export function getGCPMCPEndpoint(resource: any): string {
  const host = resource.spec?.endpoint?.host || resource.spec?.controlPlaneEndpoint?.host
  const port = resource.spec?.endpoint?.port || resource.spec?.controlPlaneEndpoint?.port
  if (host) return port && port !== 443 ? `${host}:${port}` : host
  return '-'
}

// ============================================================================
// GCPManagedMachinePool
// ============================================================================

export function getGCPMMPStatus(resource: any): StatusBadge {
  return getCAPIReadyStatus(resource)
}

export function getGCPMMPNodePoolName(resource: any): string {
  return resource.spec?.nodePoolName || resource.metadata?.name || '-'
}

export function getGCPMMPMachineType(resource: any): string {
  return resource.spec?.machineType || resource.spec?.instanceType || 'e2-medium'
}

export function getGCPMMPDiskInfo(resource: any): string {
  const diskType = resource.spec?.diskType || 'pd-standard'
  const diskSize = resource.spec?.diskSizeGb || resource.spec?.diskSizeGB || 100
  return `${diskType} ${diskSize}GB`
}

export function getGCPMMPScaling(resource: any): { min: number; max: number; autoscaling: boolean } {
  const scaling = resource.spec?.scaling || {}
  return {
    min: scaling.minCount ?? 0,
    max: scaling.maxCount ?? 0,
    autoscaling: scaling.enableAutoscaling !== false,
  }
}

export function getGCPMMPReplicas(resource: any): string {
  const ready = resource.status?.replicas ?? 0
  return String(ready)
}

export function getGCPMMPImageType(resource: any): string {
  return resource.spec?.imageType || '-'
}

// ============================================================================
// GCPMachine
// ============================================================================

export function getGCPMachineStatus(resource: any): StatusBadge {
  return getCAPIReadyStatus(resource)
}

export function getGCPMachineInstanceType(resource: any): string {
  return resource.spec?.instanceType || '-'
}

export function getGCPMachineZone(resource: any): string {
  return resource.spec?.zone || resource.spec?.failureDomain || '-'
}

export function getGCPMachineInstanceID(resource: any): string {
  return resource.status?.instanceID || resource.spec?.providerID || '-'
}

// ============================================================================
// GCPMachineTemplate
// ============================================================================

export function getGCPMTInstanceType(resource: any): string {
  return resource.spec?.template?.spec?.instanceType || '-'
}

// ============================================================================
// GCPManagedCluster
// ============================================================================

export function getGCPManagedClusterStatus(resource: any): StatusBadge {
  return getCAPIReadyStatus(resource)
}

export function getGCPManagedClusterEndpoint(resource: any): string {
  const host = resource.spec?.controlPlaneEndpoint?.host
  const port = resource.spec?.controlPlaneEndpoint?.port
  if (host) return port && port !== 443 ? `${host}:${port}` : host
  return '-'
}

export function getGCPManagedClusterProject(resource: any): string {
  return resource.spec?.project || '-'
}
