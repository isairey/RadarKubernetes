import { Cpu, Server } from 'lucide-react'
import { Section, PropertyList, Property } from '../../ui/drawer-components'
import { getAWSMTInstanceType } from '../resource-utils-aws-capi'

interface Props {
  data: any
}

export function AWSMachineTemplateRenderer({ data }: Props) {
  const templateSpec = data.spec?.template?.spec || {}
  const status = data.status || {}

  return (
    <>
      <Section title="Template Spec" icon={Cpu}>
        <PropertyList>
          <Property label="Instance Type" value={getAWSMTInstanceType(data)} />
          {templateSpec.subnet?.id && (
            <Property label="Subnet" value={<span className="font-mono text-[11px]">{templateSpec.subnet.id}</span>} />
          )}
          {templateSpec.iamInstanceProfile && <Property label="IAM Profile" value={templateSpec.iamInstanceProfile} />}
          {templateSpec.sshKeyName && <Property label="SSH Key" value={templateSpec.sshKeyName} />}
        </PropertyList>
      </Section>

      {/* Resolved capacity from status */}
      {(status.capacity || status.nodeInfo) && (
        <Section title="Resolved Info" icon={Server}>
          <PropertyList>
            {status.capacity?.cpu && <Property label="CPU" value={status.capacity.cpu} />}
            {status.capacity?.memory && <Property label="Memory" value={status.capacity.memory} />}
            {status.nodeInfo?.architecture && <Property label="Architecture" value={status.nodeInfo.architecture} />}
            {status.nodeInfo?.operatingSystem && <Property label="OS" value={status.nodeInfo.operatingSystem} />}
          </PropertyList>
        </Section>
      )}
    </>
  )
}
