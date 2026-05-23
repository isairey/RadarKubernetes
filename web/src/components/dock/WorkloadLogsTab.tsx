import { WorkloadLogsViewer } from '../logs/WorkloadLogsViewer'

interface WorkloadLogsTabProps {
  namespace: string
  workloadKind: string
  workloadName: string
}

export function WorkloadLogsTab({
  namespace,
  workloadKind,
  workloadName,
}: WorkloadLogsTabProps) {
  return (
    <div className="h-full">
      <WorkloadLogsViewer
        kind={workloadKind}
        namespace={namespace}
        name={workloadName}
      />
    </div>
  )
}
