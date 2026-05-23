// Azure CAPI Infrastructure Provider utility functions

import type { StatusBadge } from './resource-utils'
import { getCAPIReadyStatus } from './resource-utils-capi'

// ============================================================================
// AzureManagedControlPlane (AKS)
// ============================================================================

export function getAzureMCPStatus(resource: any): StatusBadge {
  return getCAPIReadyStatus(resource)
}

export function getAzureMCPLocation(resource: any): string {
  return resource.spec?.location || '-'
}

export function getAzureMCPVersion(resource: any): string {
  return resource.spec?.version || '-'
}

export function getAzureMCPResourceGroup(resource: any): string {
  return resource.spec?.resourceGroupName || '-'
}

export function getAzureMCPSKUTier(resource: any): string {
  return resource.spec?.sku?.tier || '-'
}

export function getAzureMCPNetworkPlugin(resource: any): string {
  return resource.spec?.networkPlugin || '-'
}

export function getAzureMCPNetworkPolicy(resource: any): string {
  return resource.spec?.networkPolicy || '-'
}

export function getAzureMCPDNSPrefix(resource: any): string {
  return resource.spec?.dnsPrefix || '-'
}

export function getAzureMCPUpgradeChannel(resource: any): string {
  return resource.spec?.autoUpgradeProfile?.upgradeChannel || '-'
}

export function getAzureMCPPrivateCluster(resource: any): boolean {
  return !!resource.spec?.apiServerAccessProfile?.enablePrivateCluster
}

// ============================================================================
// AzureManagedMachinePool (AKS node pool)
// ============================================================================

export function getAzureMMPStatus(resource: any): StatusBadge {
  return getCAPIReadyStatus(resource)
}

export function getAzureMMPSKU(resource: any): string {
  return resource.spec?.sku || '-'
}

export function getAzureMMPMode(resource: any): string {
  return resource.spec?.mode || '-'
}

export function getAzureMMPOSDiskInfo(resource: any): string {
  const diskType = resource.spec?.osDiskType || 'Managed'
  const diskSize = resource.spec?.osDiskSizeGB
  if (diskSize) return `${diskType} ${diskSize}GB`
  return diskType
}

export function getAzureMMPScaling(resource: any): { min: number; max: number } {
  const scaling = resource.spec?.scaling || {}
  return {
    min: scaling.minSize ?? 0,
    max: scaling.maxSize ?? 0,
  }
}

export function getAzureMMPReplicas(resource: any): string {
  return String(resource.status?.replicas ?? 0)
}

export function getAzureMMPScaleSetPriority(resource: any): string {
  return resource.spec?.scaleSetPriority || 'Regular'
}

export function getAzureMMPOSType(resource: any): string {
  return resource.spec?.osType || 'Linux'
}

// ============================================================================
// AzureMachine
// ============================================================================

export function getAzureMachineStatus(resource: any): StatusBadge {
  return getCAPIReadyStatus(resource)
}

export function getAzureMachineVMSize(resource: any): string {
  return resource.spec?.vmSize || '-'
}

export function getAzureMachineProviderID(resource: any): string {
  return resource.spec?.providerID || '-'
}

// ============================================================================
// AzureMachineTemplate
// ============================================================================

export function getAzureMTVMSize(resource: any): string {
  return resource.spec?.template?.spec?.vmSize || '-'
}

// ============================================================================
// AzureManagedCluster
// ============================================================================

export function getAzureManagedClusterStatus(resource: any): StatusBadge {
  return getCAPIReadyStatus(resource)
}
