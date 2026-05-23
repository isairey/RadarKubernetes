// GCP CAPI Infrastructure Provider cell components for ResourcesView table

import { Tooltip } from '../../ui/Tooltip'
import { clsx } from 'clsx'
import {
  getGCPMCPStatus, getGCPMCPClusterName, getGCPMCPProject, getGCPMCPLocation, getGCPMCPVersion,
  getGCPMMPStatus, getGCPMMPMachineType, getGCPMMPReplicas,
  getGCPMachineStatus, getGCPMachineInstanceType, getGCPMachineZone,
  getGCPMTInstanceType,
  getGCPManagedClusterStatus, getGCPManagedClusterProject,
} from '../resource-utils-gcp-capi'

function StatusBadge({ resource, getStatus }: { resource: any; getStatus: (r: any) => { text: string; color: string } }) {
  const status = getStatus(resource)
  return (
    <Tooltip content={status.text}>
      <span className={clsx('badge truncate max-w-[140px]', status.color)}>{status.text}</span>
    </Tooltip>
  )
}

function TextCell({ value }: { value: string }) {
  return <span className="text-sm text-theme-text-secondary">{value}</span>
}

export function GCPManagedControlPlaneCell({ resource, column }: { resource: any; column: string }) {
  switch (column) {
    case 'status': return <StatusBadge resource={resource} getStatus={getGCPMCPStatus} />
    case 'gkeCluster': return <TextCell value={getGCPMCPClusterName(resource)} />
    case 'project': return <TextCell value={getGCPMCPProject(resource)} />
    case 'location': return <TextCell value={getGCPMCPLocation(resource)} />
    case 'version': return <TextCell value={getGCPMCPVersion(resource)} />
    default: return <span className="text-sm text-theme-text-tertiary">-</span>
  }
}

export function GCPManagedMachinePoolCell({ resource, column }: { resource: any; column: string }) {
  switch (column) {
    case 'status': return <StatusBadge resource={resource} getStatus={getGCPMMPStatus} />
    case 'machineType': return <TextCell value={getGCPMMPMachineType(resource)} />
    case 'replicas': return <TextCell value={getGCPMMPReplicas(resource)} />
    default: return <span className="text-sm text-theme-text-tertiary">-</span>
  }
}

export function GCPMachineCell({ resource, column }: { resource: any; column: string }) {
  switch (column) {
    case 'status': return <StatusBadge resource={resource} getStatus={getGCPMachineStatus} />
    case 'instanceType': return <TextCell value={getGCPMachineInstanceType(resource)} />
    case 'zone': return <TextCell value={getGCPMachineZone(resource)} />
    default: return <span className="text-sm text-theme-text-tertiary">-</span>
  }
}

export function GCPMachineTemplateCell({ resource, column }: { resource: any; column: string }) {
  switch (column) {
    case 'instanceType': return <TextCell value={getGCPMTInstanceType(resource)} />
    default: return <span className="text-sm text-theme-text-tertiary">-</span>
  }
}

export function GCPManagedClusterCell({ resource, column }: { resource: any; column: string }) {
  switch (column) {
    case 'status': return <StatusBadge resource={resource} getStatus={getGCPManagedClusterStatus} />
    case 'project': return <TextCell value={getGCPManagedClusterProject(resource)} />
    default: return <span className="text-sm text-theme-text-tertiary">-</span>
  }
}
