import { useEffect, useState } from 'react'
import { api } from '../api'
import type { Project } from '../types'
import { ProjectMetadataPanel } from '../components/ProjectMetadataPanel'
import { RuntimeStatusBadge } from '../components/RuntimeStatusBadge'
import { Button } from '../components/ui/button'
import { ExternalLink, Play, RotateCcw, Square, Trash2 } from 'lucide-react'
import { useProjects } from '../store'

export function ProjectDetail({ id }: { id: string }) {
  const { select, remove, start, stop } = useProjects()
  const [project, setProject] = useState<Project | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const reload = async () => {
    setError(null)
    try {
      setProject(await api.getProject(id))
    } catch (e: any) {
      setError(e?.message ?? String(e))
    }
  }

  // Reload whenever the selected project changes
  useEffect(() => {
    setProject(null)
    setError(null)
    reload()
  }, [id])

  if (error) {
    return (
      <div className="h-full flex items-center justify-center p-8">
        <div className="glass rounded-2xl px-6 py-5 max-w-md text-center space-y-3">
          <p className="text-sm font-medium text-destructive">Failed to load project</p>
          <p className="text-xs text-[var(--orbit-muted)]">{error}</p>
          <Button variant="ghost" size="sm" onClick={reload}>
            <RotateCcw className="h-3.5 w-3.5" />
            Retry
          </Button>
        </div>
      </div>
    )
  }

  if (!project) {
    return (
      <div className="h-full flex items-center justify-center">
        <p className="text-sm text-[var(--orbit-muted)]">Loading…</p>
      </div>
    )
  }

  const isRunning = project.status === 'running' || project.status === 'starting'

  const onStart = async () => {
    setBusy(true)
    try {
      await start(project.id)
      await reload()
    } finally {
      setBusy(false)
    }
  }

  const onStop = async () => {
    setBusy(true)
    try {
      await stop(project.id)
      await reload()
    } finally {
      setBusy(false)
    }
  }

  const onDelete = async () => {
    if (!confirm(`Remove "${project.name}" from Orbit?`)) return
    await remove(project.id)
    select(null)
  }

  return (
    <div className="h-full overflow-auto scrollbar-thin">
      <div className="p-8 space-y-6 max-w-5xl">

        {/* Header */}
        <div className="flex items-end justify-between gap-4">
          <div className="min-w-0">
            <div className="mb-2">
              <RuntimeStatusBadge status={project.status} />
            </div>
            <h1 className="text-2xl font-semibold tracking-tight truncate">
              {project.name}
            </h1>
            <p className="mt-1 text-xs text-[var(--orbit-muted)] font-mono truncate">
              {project.path}
            </p>
            <a
              className="mt-1 text-sm text-[var(--orbit-accent)] inline-flex items-center gap-1.5 hover:underline"
              href={`https://${project.localDomain}`}
              target="_blank"
              rel="noreferrer"
            >
              https://{project.localDomain}
              <ExternalLink className="h-3 w-3" />
            </a>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            {isRunning ? (
              <Button variant="secondary" onClick={onStop} disabled={busy}>
                <Square />
                Stop
              </Button>
            ) : (
              <Button onClick={onStart} disabled={busy}>
                <Play />
                Start
              </Button>
            )}
            <Button variant="destructive" onClick={onDelete} disabled={busy}>
              <Trash2 />
              Remove
            </Button>
          </div>
        </div>

        {/* Body */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Metadata */}
          <div className="lg:col-span-2">
            <ProjectMetadataPanel project={project} />
          </div>

          {/* Scripts */}
          <div className="glass rounded-2xl p-4 space-y-3">
            <h3 className="text-sm font-semibold tracking-tight">Scripts</h3>
            {(project.scripts ?? []).length === 0 ? (
              <p className="text-xs text-[var(--orbit-muted)]">No scripts detected.</p>
            ) : (
              <div className="space-y-1.5">
                {(project.scripts ?? []).map((s) => (
                  <div
                    key={s.id}
                    className="rounded-lg bg-white/[0.04] border border-[var(--orbit-border)] px-3 py-2"
                  >
                    <div className="text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)]">
                      {s.name}
                    </div>
                    <div className="font-mono text-xs truncate mt-0.5">{s.command}</div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

      </div>
    </div>
  )
}
