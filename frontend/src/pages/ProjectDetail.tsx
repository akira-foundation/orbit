import { useEffect, useState } from 'react'
import { api } from '../api'
import type { Project } from '../types'
import { ProjectMetadataPanel } from '../components/ProjectMetadataPanel'
import { RuntimeStatusBadge } from '../components/RuntimeStatusBadge'
import { Button } from '../components/Button'
import { ArrowLeft, ExternalLink, Play, Square, Trash2 } from 'lucide-react'
import { useProjects } from '../store'

export function ProjectDetail({ id }: { id: string }) {
  const { select, remove, start, stop } = useProjects()
  const [project, setProject] = useState<Project | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const reload = async () => {
    try { setProject(await api.getProject(id)) }
    catch (e: any) { setError(e?.message ?? String(e)) }
  }

  useEffect(() => { reload() }, [id])

  if (error) return <div className="text-rose-300 text-sm">{error}</div>
  if (!project) return <div className="text-[var(--color-muted)] text-sm">Loading…</div>

  const onStart = async () => { setBusy(true); try { await start(project.id); await reload() } finally { setBusy(false) } }
  const onStop  = async () => { setBusy(true); try { await stop(project.id); await reload() } finally { setBusy(false) } }
  const onDelete = async () => {
    if (!confirm(`Remove ${project.name} from Orbit?`)) return
    await remove(project.id)
    select(null)
  }

  const isRunning = project.status === 'running' || project.status === 'starting'

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <button onClick={() => select(null)} className="inline-flex items-center gap-2 text-sm text-[var(--color-muted)] hover:text-white">
          <ArrowLeft className="h-4 w-4" />Back
        </button>
        <RuntimeStatusBadge status={project.status} />
      </div>

      <div className="flex items-end justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{project.name}</h1>
          <a className="text-sm text-[var(--color-accent-2)] inline-flex items-center gap-1.5" href={`https://${project.localDomain}`} target="_blank" rel="noreferrer">
            https://{project.localDomain} <ExternalLink className="h-3 w-3" />
          </a>
        </div>
        <div className="flex gap-2">
          {isRunning
            ? <Button onClick={onStop}  disabled={busy} variant="secondary"><Square className="h-4 w-4" />Stop</Button>
            : <Button onClick={onStart} disabled={busy} variant="primary"><Play className="h-4 w-4" />Start</Button>}
          <Button onClick={onDelete} variant="danger"><Trash2 className="h-4 w-4" />Remove</Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <ProjectMetadataPanel project={project} />
        </div>

        <div className="rounded-xl bg-[var(--color-surface)] ring-1 ring-[var(--color-border)] p-4">
          <h3 className="font-semibold tracking-tight mb-3">Scripts</h3>
          <div className="space-y-2">
            {(project.scripts ?? []).length === 0 && <p className="text-sm text-[var(--color-muted)]">No scripts detected.</p>}
            {(project.scripts ?? []).map(s => (
              <div key={s.id} className="rounded-lg bg-[var(--color-bg)]/40 ring-1 ring-[var(--color-border)] px-3 py-2">
                <div className="text-xs text-[var(--color-muted)]">{s.name}</div>
                <div className="font-mono text-xs truncate">{s.command}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
