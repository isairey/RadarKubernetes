// AWS CAPI Infrastructure Provider utility functions

import type { StatusBadge } from './resource-utils'
import { healthColors } from './resource-utils'
import { getCAPIConditions, getCAPIReadyStatus } from './resource-utils-capi'

// ============================================================================
// AWSManagedControlPlane
// ============================================================================

export function getAWSMCPStatus(resource: any): StatusBadge {
  return getCAPIReadyStatus(resource)
}

export function getAWSMCPEKSClusterName(resource: any): string {
  return resource.spec?.eksClusterName || '-'
}

export function getAWSMCPRegion(resource: any): string {
  return resource.spec?.region || '-'
}

export function getAWSMCPVersion(resource: any): string {
  return resource.spec?.version || '-'
}

export function getAWSMCPEndpointAccess(resource: any): string {
  const pub = resource.spec?.endpointAccess?.public
  const priv = resource.spec?.endpointAccess?.private
  if (pub && priv) return 'Public & Private'
  if (pub) return 'Public'
  if (priv) return 'Private'
  return '-'
}

export interface AWSAddon {
  name: string
  specVersion: string
  statusVersion: string
  status: string
  arn: string
}

export function getAWSMCPAddons(resource: any): AWSAddon[] {
  const specAddons = resource.spec?.addons || []
  const statusAddons = resource.status?.addons || []
  const statusMap = new Map(statusAddons.map((a: any) => [a.name, a]))

  return specAddons.map((sa: any) => {
    const st: any = statusMap.get(sa.name) || {}
    return {
      name: sa.name,
      specVersion: sa.version || '-',
      statusVersion: st.currentVersion || st.version || '-',
      status: st.status || 'Unknown',
      arn: st.arn || '-',
    }
  })
}

export interface AWSSubnet {
  id: string
  az: string
  isPublic: boolean
  cidrBlock: string
}

export function getAWSMCPSubnets(resource: any): AWSSubnet[] {
  const subnets = resource.spec?.network?.subnets || []
  return subnets.map((s: any) => ({
    id: s.id || s.resourceID || '-',
    az: s.availabilityZone || '-',
    isPublic: !!s.isPublic,
    cidrBlock: s.cidrBlock || '-',
  }))
}

export interface AWSSecurityGroup {
  role: string
  id: string
  name: string
}

export function getAWSMCPSecurityGroups(resource: any): AWSSecurityGroup[] {
  const sgs = resource.status?.networkStatus?.securityGroups || {}
  const result: AWSSecurityGroup[] = []
  for (const [role, sg] of Object.entries(sgs) as [string, any][]) {
    result.push({ role, id: sg.id || '-', name: sg.name || '-' })
  }
  return result
}

export function getAWSMCPNATGatewayIPs(resource: any): string[] {
  return resource.status?.networkStatus?.natGatewaysIPs || []
}

export function getAWSMCPFailureDomains(resource: any): string[] {
  const fd = resource.status?.failureDomains || {}
  return Object.keys(fd).sort()
}

export function getAWSMCPVPC(resource: any): { id: string; cidrBlock: string } {
  const vpc = resource.spec?.network?.vpc || {}
  return { id: vpc.id || '-', cidrBlock: vpc.cidrBlock || '-' }
}

// ============================================================================
// AWSManagedMachinePool
// ============================================================================

export function getAWSMMPStatus(resource: any): StatusBadge {
  return getCAPIReadyStatus(resource)
}

export function getAWSMMPInstanceType(resource: any): string {
  return resource.spec?.instanceType || '-'
}

export function getAWSMMPCapacityType(resource: any): string {
  return resource.spec?.capacityType || '-'
}

export function getAWSMMPAMIType(resource: any): string {
  return resource.spec?.amiType || '-'
}

export function getAWSMMPNodegroupName(resource: any): string {
  return resource.spec?.eksNodegroupName || '-'
}

export function getAWSMMPScaling(resource: any): { min: number; max: number } {
  return {
    min: resource.spec?.scaling?.minSize ?? 0,
    max: resource.spec?.scaling?.maxSize ?? 0,
  }
}

export function getAWSMMPReplicas(resource: any): string {
  const ready = resource.status?.replicas ?? 0
  const desired = resource.spec?.scaling?.maxSize
  if (desired != null) return `${ready}/${desired}`
  return String(ready)
}

// ============================================================================
// AWSMachine
// ============================================================================

export function getAWSMachineStatus(resource: any): StatusBadge {
  const state = resource.status?.instanceState?.toLowerCase()
  if (state === 'running') {
    const conditions = getCAPIConditions(resource)
    const readyCond = conditions.find((c: any) => c.type === 'Ready')
    if (readyCond?.status === 'True') return { text: 'Running', color: healthColors.healthy, level: 'healthy' }
    if (readyCond?.status === 'False') return { text: readyCond.reason || 'NotReady', color: healthColors.unhealthy, level: 'unhealthy' }
    return { text: 'Running', color: healthColors.healthy, level: 'healthy' }
  }
  if (state === 'pending') return { text: 'Pending', color: healthColors.degraded, level: 'degraded' }
  if (state === 'terminated' || state === 'shutting-down') return { text: state, color: healthColors.unhealthy, level: 'unhealthy' }
  if (state === 'stopping' || state === 'stopped') return { text: state, color: healthColors.degraded, level: 'degraded' }
  return getCAPIReadyStatus(resource)
}

export function getAWSMachineInstanceType(resource: any): string {
  return resource.spec?.instanceType || '-'
}

export function getAWSMachineInstanceState(resource: any): string {
  return resource.status?.instanceState || '-'
}

export function getAWSMachineInstanceID(resource: any): string {
  return resource.spec?.instanceID || '-'
}

// ============================================================================
// AWSMachineTemplate
// ============================================================================

export function getAWSMTInstanceType(resource: any): string {
  return resource.spec?.template?.spec?.instanceType || '-'
}

export function getAWSMTCapacity(resource: any): string {
  const cap = resource.status?.capacity
  if (!cap) return '-'
  const parts: string[] = []
  if (cap.cpu) parts.push(`${cap.cpu} CPU`)
  if (cap.memory) parts.push(cap.memory)
  return parts.join(', ') || '-'
}

// ============================================================================
// AWSManagedCluster
// ============================================================================

export function getAWSManagedClusterStatus(resource: any): StatusBadge {
  return getCAPIReadyStatus(resource)
}

export function getAWSManagedClusterEndpoint(resource: any): string {
  const host = resource.spec?.controlPlaneEndpoint?.host
  const port = resource.spec?.controlPlaneEndpoint?.port
  if (host) return port && port !== 443 ? `${host}:${port}` : host
  return '-'
}

export function getAWSManagedClusterFailureDomains(resource: any): string[] {
  const fd = resource.status?.failureDomains || {}
  return Object.keys(fd).sort()
}
