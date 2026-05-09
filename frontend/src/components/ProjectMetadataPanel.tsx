import type { Project } from '../types'

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
  return (
    <div className="glass rounded-2xl divide-y divide-[var(--orbit-border)]">
      {items.map(([k, v]) => (
        <div key={k} className="flex items-center justify-between gap-6 px-4 py-3 text-sm">
          <span className="text-[var(--orbit-muted)]">{k}</span>
          <span className="font-mono text-xs text-right truncate">{v}</span>
        </div>
      ))}
    </div>
  )
}
