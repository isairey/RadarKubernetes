import { Settings } from 'lucide-react'
import { Section, PropertyList, Property, ConditionsSection } from '../../ui/drawer-components'

interface Props {
  data: any
}

export function CAPIMachineDrainRuleRenderer({ data }: Props) {
  const spec = data.spec || {}
  const conditions = data.status?.v1beta2?.conditions || data.status?.conditions || []

  const machines = spec.machines || []
  const drain = spec.drain || {}

  return (
    <>
      <Section title="Drain Configuration" icon={Settings}>
        <PropertyList>
          {drain.behavior && <Property label="Behavior" value={drain.behavior} />}
          {drain.order != null && <Property label="Order" value={String(drain.order)} />}
        </PropertyList>
      </Section>

      {machines.length > 0 && (
        <Section title="Machine Selectors" icon={Settings}>
          {machines.map((m: any, i: number) => (
            <div key={i} className="text-xs text-theme-text-secondary mb-1">
              {m.clusterName && <span>Cluster: {m.clusterName} </span>}
              {m.namespace && <span>NS: {m.namespace} </span>}
            </div>
          ))}
        </Section>
      )}

      <ConditionsSection conditions={conditions} />
    </>
  )
}
