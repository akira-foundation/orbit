import type { Project, ProjectProcess } from '../types'

function formatProcess(p: ProjectProcess): string {
  const label = p.role === 'backend' ? 'Backend' : 'Frontend'
  const detail = p.kind === 'php-fpm' ? `php-fpm ${p.phpVersion}` : p.command
  const dir = p.workDir ? `${p.workDir}/` : ''
  return `${label}: ${dir}${detail}`
}

export function ProjectMetadataPanel({ project }: { project: Project }) {
  const items: Array<[string, string]> = [
    ['Framework', project.detectedFramework],
    ['Package manager', project.packageManager],
    ['Dev command', project.devCommand],
    ['Dev port', project.devPort ? String(project.devPort) : '—'],
    ['Local domain', `https://${project.localDomain}`],
    ['Path', project.path],
    ['Slug', project.slug],
    ['Created', new Date(project.createdAt).toLocaleString()],
  ]
  const processes = project.processes ?? []
  return (
    <div className="glass rounded-2xl divide-y divide-[var(--orbit-border)]">
      {items.map(([k, v]) => (
        <div key={k} className="flex items-center justify-between gap-6 px-4 py-3 text-sm">
          <span className="text-[var(--orbit-muted)]">{k}</span>
          <span className="font-mono text-xs text-right truncate">{v}</span>
        </div>
      ))}
      {processes.length > 1 && (
        <div className="flex items-start justify-between gap-6 px-4 py-3 text-sm">
          <span className="text-[var(--orbit-muted)]">Processes</span>
          <div className="text-right space-y-1">
            {processes.map((p) => (
              <div key={p.id} className="font-mono text-xs truncate">{formatProcess(p)}</div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
