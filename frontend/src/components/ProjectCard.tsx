import type { Project } from '../types'
import { RuntimeStatusBadge } from './RuntimeStatusBadge'
import { Box, Folder, Globe } from 'lucide-react'

export function ProjectCard({ project, onOpen }: { project: Project; onOpen: (p: Project) => void }) {
  return (
    <button
      onClick={() => onOpen(project)}
      className="group text-left rounded-xl bg-[var(--color-surface)] ring-1 ring-[var(--color-border)] p-5 hover:ring-[var(--color-accent)]/40 hover:bg-[var(--color-surface-2)] transition"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <Box className="h-4 w-4 text-[var(--color-muted)]" />
            <h3 className="font-semibold tracking-tight truncate">{project.name}</h3>
          </div>
          <div className="mt-1 flex items-center gap-1.5 text-xs text-[var(--color-muted)]">
            <Globe className="h-3 w-3" />
            <span className="truncate">https://{project.localDomain}</span>
          </div>
        </div>
        <RuntimeStatusBadge status={project.status} />
      </div>

      <div className="mt-4 grid grid-cols-2 gap-2 text-xs">
        <Meta label="Framework" value={project.detectedFramework} />
        <Meta label="Manager"   value={project.packageManager} />
        <Meta label="Command"   value={project.devCommand} />
        <Meta label="Port"      value={project.devPort ? String(project.devPort) : '—'} />
      </div>

      <div className="mt-4 flex items-center gap-1.5 text-[11px] text-[var(--color-muted)]">
        <Folder className="h-3 w-3" />
        <span className="truncate">{project.path}</span>
      </div>
    </button>
  )
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md bg-[var(--color-bg)]/40 ring-1 ring-[var(--color-border)] px-2.5 py-1.5">
      <div className="text-[10px] uppercase tracking-widest text-[var(--color-muted)]">{label}</div>
      <div className="truncate text-[var(--color-text)]">{value}</div>
    </div>
  )
}
