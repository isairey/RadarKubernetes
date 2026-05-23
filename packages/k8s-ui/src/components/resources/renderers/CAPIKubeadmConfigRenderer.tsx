import { Settings, FileText } from 'lucide-react'
import { Section, PropertyList, Property, ConditionsSection } from '../../ui/drawer-components'

interface Props {
  data: any
}

export function CAPIKubeadmConfigRenderer({ data }: Props) {
  const status = data.status || {}
  const spec = data.spec || {}
  const conditions = status.v1beta2?.conditions || status.conditions || []

  const clusterConfig = spec.clusterConfiguration || {}
  const files = spec.files || []
  const preKubeadmCommands = spec.preKubeadmCommands || []
  const postKubeadmCommands = spec.postKubeadmCommands || []
  const certSANs = clusterConfig.certSANs || []

  return (
    <>
      <Section title="Overview" icon={Settings}>
        <PropertyList>
          {status.ready != null && <Property label="Ready" value={status.ready ? 'Yes' : 'No'} />}
          {status.dataSecretName && <Property label="Data Secret" value={status.dataSecretName} />}
        </PropertyList>
      </Section>

      {certSANs.length > 0 && (
        <Section title="Cert SANs" icon={Settings}>
          <div className="flex flex-wrap gap-1">
            {certSANs.map((san: string, i: number) => (
              <span key={i} className="badge badge-sm bg-theme-surface text-theme-text-secondary border-theme-border">{san}</span>
            ))}
          </div>
        </Section>
      )}

      {/* API Server Extra Args */}
      {clusterConfig.apiServer?.extraArgs && Object.keys(clusterConfig.apiServer.extraArgs).length > 0 && (
        <Section title="API Server Extra Args" icon={Settings} defaultExpanded={false}>
          <PropertyList>
            {Object.entries(clusterConfig.apiServer.extraArgs).map(([key, value]) => (
              <Property key={key} label={key} value={String(value)} />
            ))}
          </PropertyList>
        </Section>
      )}

      {/* Files */}
      {files.length > 0 && (
        <Section title="Files" icon={FileText} defaultExpanded={false}>
          <table className="w-full text-xs">
            <thead>
              <tr className="text-theme-text-tertiary">
                <th className="text-left font-medium py-1">Path</th>
                <th className="text-left font-medium py-1">Owner</th>
                <th className="text-left font-medium py-1">Permissions</th>
              </tr>
            </thead>
            <tbody>
              {files.map((f: any, i: number) => (
                <tr key={i} className="border-t border-theme-border">
                  <td className="py-1 text-theme-text-secondary font-mono text-[10px]">{f.path}</td>
                  <td className="py-1 text-theme-text-secondary">{f.owner || '-'}</td>
                  <td className="py-1 text-theme-text-secondary">{f.permissions || '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Section>
      )}

      {/* Commands */}
      {preKubeadmCommands.length > 0 && (
        <Section title="Pre-Kubeadm Commands" icon={Settings} defaultExpanded={false}>
          <div className="text-xs font-mono text-theme-text-secondary space-y-0.5">
            {preKubeadmCommands.map((cmd: string, i: number) => (
              <div key={i} className="truncate">{cmd}</div>
            ))}
          </div>
        </Section>
      )}

      {postKubeadmCommands.length > 0 && (
        <Section title="Post-Kubeadm Commands" icon={Settings} defaultExpanded={false}>
          <div className="text-xs font-mono text-theme-text-secondary space-y-0.5">
            {postKubeadmCommands.map((cmd: string, i: number) => (
              <div key={i} className="truncate">{cmd}</div>
            ))}
          </div>
        </Section>
      )}

      <ConditionsSection conditions={conditions} />
    </>
  )
}
