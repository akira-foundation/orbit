import { useEffect, useState } from "react"
import { api } from "../api"
import type { Project } from "../types"
import { ProjectMetadataPanel } from "../components/ProjectMetadataPanel"
import { RuntimeStatusBadge } from "../components/RuntimeStatusBadge"
import { Button } from "../components/ui/button"
import { ExternalLink, Play, Square, Trash2 } from "lucide-react"
import { useProjects } from "../store"

export function ProjectDetail({ id }: { id: string }) {
  const { select, remove, start, stop } = useProjects()
  const [project, setProject] = useState<Project | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const reload = async () => {
    try {
      setProject(await api.getProject(id))
    } catch (e: any) {
      setError(e?.message ?? String(e))
    }
  }

  useEffect(() => {
    reload()
  }, [id])

  if (error) return <div className="p-8 text-destructive text-sm">{error}</div>
  if (!project)
    return <div className="p-8 text-[var(--orbit-muted)] text-sm">Loading…</div>

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
    if (!confirm(`Remove ${project.name} from Orbit?`)) return
    await remove(project.id)
    select(null)
  }

  const isRunning =
    project.status === "running" || project.status === "starting"

  return (
    <div className="h-full overflow-auto scrollbar-thin">
      <div className="p-8 space-y-6">
        <div className="flex items-end justify-between gap-4">
          <div>
            <div className="mb-1">
              <RuntimeStatusBadge status={project.status} />
            </div>
            <h1 className="text-2xl font-semibold tracking-tight">
              {project.name}
            </h1>
            <a
              className="text-sm text-[var(--orbit-accent)] inline-flex items-center gap-1.5"
              href={`https://${project.localDomain}`}
              target="_blank"
              rel="noreferrer"
            >
              https://{project.localDomain}{" "}
              <ExternalLink className="h-3 w-3" />
            </a>
          </div>
          <div className="flex gap-2">
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
            <Button variant="destructive" onClick={onDelete}>
              <Trash2 />
              Remove
            </Button>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2">
            <ProjectMetadataPanel project={project} />
          </div>

          <div className="glass rounded-2xl p-4">
            <h3 className="font-semibold tracking-tight mb-3">Scripts</h3>
            <div className="space-y-2">
              {(project.scripts ?? []).length === 0 && (
                <p className="text-sm text-[var(--orbit-muted)]">
                  No scripts detected.
                </p>
              )}
              {(project.scripts ?? []).map((s) => (
                <div
                  key={s.id}
                  className="rounded-lg bg-white/[0.04] border border-[var(--orbit-border)] px-3 py-2"
                >
                  <div className="text-xs text-[var(--orbit-muted)]">
                    {s.name}
                  </div>
                  <div className="font-mono text-xs truncate">{s.command}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
