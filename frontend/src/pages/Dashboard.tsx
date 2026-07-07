import { useEffect, useMemo, useState } from 'react'
import { useProjects } from '../store'
import { BarChart3, ExternalLink, FileText, GitBranch, Loader2, Orbit, Play, Plus, Square } from 'lucide-react'
import { Button } from '../components/ui/button'
import type { GitInfo, Project } from '../types'
import { cn } from '../lib/cn'
import { RuntimeStatusBadge } from '../components/RuntimeStatusBadge'
import { PROJECT_DND_MIME } from '../components/SidebarGroups'
import { api } from '../api'

export function Dashboard({ onAdd }: { onAdd: () => void }) {
  const { projects, filter, query, select } = useProjects()
  const [focusId, setFocusId] = useState<string | null>(null)

  useEffect(() => {
    setFocusId(null)
  }, [filter, query])

  const filtered = useMemo(() => {
    let xs = projects
    if (filter !== 'all') xs = xs.filter((p) => p.status === filter)
    if (query.trim()) {
      const q = query.toLowerCase()
      xs = xs.filter(
        (p) =>
          p.name.toLowerCase().includes(q) ||
          p.detectedFramework.toLowerCase().includes(q) ||
          p.localDomain.toLowerCase().includes(q),
      )
    }
    return xs
  }, [projects, filter, query])

  if (projects.length === 0) return <Empty onAdd={onAdd} />

  if (filtered.length === 0) {
    return (
      <div className="h-full flex items-center justify-center">
        <p className="text-sm text-[var(--orbit-muted)]">No projects match.</p>
      </div>
    )
  }

  return (
    <div className="h-full overflow-auto scrollbar-thin" onClick={() => setFocusId(null)}>
      <ListView
        projects={filtered}
        focusId={focusId}
        setFocusId={setFocusId}
        onOpen={(p) => { setFocusId(null); select(p.id) }}
      />
    </div>
  )
}

function ListView({
  projects,
  focusId,
  setFocusId,
  onOpen,
}: {
  projects: Project[]
  focusId: string | null
  setFocusId: (id: string | null) => void
  onOpen: (p: Project) => void
}) {
  return (
    <div className="p-4">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-xs uppercase tracking-widest text-[var(--orbit-subtle)]">
            <Th>Name</Th>
            <Th>Status</Th>
            <Th>Framework</Th>
            <Th>Manager</Th>
            <Th>Branch</Th>
            <Th>Domain</Th>
            <Th>{''}</Th>
          </tr>
        </thead>
        <tbody>
          {projects.map((p) => (
            <tr
              key={p.id}
              draggable
              onDragStart={(e) => {
                e.dataTransfer.setData(PROJECT_DND_MIME, p.id)
                e.dataTransfer.effectAllowed = 'copy'
              }}
              onClick={(e) => { e.stopPropagation(); setFocusId(p.id) }}
              onDoubleClick={() => onOpen(p)}
              className={cn(
                'cursor-default border-t border-[var(--orbit-border)] transition-colors',
                focusId === p.id ? 'bg-white/[0.10]' : 'hover:bg-white/4',
              )}
            >
              <Td className="font-medium">{p.name}</Td>
              <Td><RuntimeStatusBadge status={p.status} /></Td>
              <Td>{p.detectedFramework}</Td>
              <Td>{p.packageManager}</Td>
              <Td><GitChip projectId={p.id} /></Td>
              <Td>
                <div className="flex items-center gap-1.5">
                  <span className="font-mono text-xs text-[var(--orbit-muted)] truncate">{p.localDomain}</span>
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      api.openProject(p.id)
                    }}
                    className="shrink-0 text-[var(--orbit-muted)] hover:text-[var(--orbit-text)] transition-colors"
                  >
                    <ExternalLink className="size-3" />
                  </button>
                </div>
              </Td>
              <Td className="w-px text-right pr-3">
                <div className="flex items-center justify-end gap-1">
                  <RowOpenButton projectId={p.id} />
                  <RowLogsButton projectId={p.id} />
                  <RowMetricsButton projectId={p.id} />
                  <RowAction project={p} />
                </div>
              </Td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function GitChip({ projectId }: { projectId: string }) {
  const [info, setInfo] = useState<GitInfo | null>(null)

  useEffect(() => {
    let alive = true
    const load = () => {
      api.gitStatus(projectId)
        .then((g) => { if (alive) setInfo(g) })
        .catch(() => { if (alive) setInfo(null) })
    }
    load()
    const onFocus = () => load()
    window.addEventListener('focus', onFocus)
    const t = setInterval(load, 10000)
    return () => {
      alive = false
      window.removeEventListener('focus', onFocus)
      clearInterval(t)
    }
  }, [projectId])

  if (!info || !info.repo) {
    return <span className="text-xs text-[var(--orbit-subtle)]">—</span>
  }
  return (
    <span className="inline-flex items-center gap-1.5 text-xs" title={
      `${info.branch}${info.dirty > 0 ? ` · ${info.dirty} uncommitted` : ' · clean'}` +
      `${info.ahead > 0 ? ` · ${info.ahead} ahead` : ''}${info.behind > 0 ? ` · ${info.behind} behind` : ''}`
    }>
      <GitBranch className="size-3 shrink-0 text-[var(--orbit-muted)]" />
      <span className="font-mono text-[var(--orbit-muted)] truncate max-w-28">{info.branch}</span>
      {info.dirty > 0 && (
        <span className="inline-flex items-center gap-1 text-[var(--orbit-accent-2)]">
          <span className="size-1.5 rounded-full bg-[var(--orbit-accent-2)]" />
          {info.dirty}
        </span>
      )}
    </span>
  )
}

function RowOpenButton({ projectId }: { projectId: string }) {
  return (
    <button
      onClick={(e) => {
        e.stopPropagation()
        api.openProject(projectId)
      }}
      title="Open in browser"
      className="no-drag inline-flex items-center justify-center size-7 rounded-md text-[var(--orbit-muted)] hover:text-[var(--orbit-accent-2)] hover:bg-white/5 transition-colors"
    >
      <ExternalLink className="size-3.5" />
    </button>
  )
}

function RowLogsButton({ projectId }: { projectId: string }) {
  const { showProjectLogs } = useProjects()
  return (
    <button
      onClick={(e) => {
        e.stopPropagation()
        showProjectLogs(projectId)
      }}
      title="Logs"
      className="no-drag inline-flex items-center justify-center size-7 rounded-md text-[var(--orbit-muted)] hover:text-amber-300 hover:bg-white/5 transition-colors"
    >
      <FileText className="size-3.5" />
    </button>
  )
}

function RowMetricsButton({ projectId }: { projectId: string }) {
  const { showProjectMetrics } = useProjects()
  return (
    <button
      onClick={(e) => {
        e.stopPropagation()
        showProjectMetrics(projectId)
      }}
      title="Metrics"
      className="no-drag inline-flex items-center justify-center size-7 rounded-md text-[var(--orbit-muted)] hover:text-cyan-300 hover:bg-white/5 transition-colors"
    >
      <BarChart3 className="size-3.5" />
    </button>
  )
}

function RowAction({ project }: { project: Project }) {
  const [busy, setBusy] = useState(false)
  const running = project.status === 'running' || project.status === 'starting'
  const onClick = async (e: React.MouseEvent) => {
    e.stopPropagation()
    if (busy) return
    setBusy(true)
    try {
      if (running) await api.stopProject(project.id)
      else await api.startProject(project.id)
    } finally {
      setBusy(false)
    }
  }
  const Icon = busy ? Loader2 : running ? Square : Play
  return (
    <button
      onClick={onClick}
      disabled={busy}
      title={running ? 'Stop' : 'Start'}
      className={cn(
        'no-drag inline-flex items-center justify-center size-7 rounded-md transition-colors',
        running
          ? 'text-rose-300 hover:bg-white/5'
          : 'text-emerald-300 hover:bg-white/5',
        busy && 'opacity-60 cursor-default',
      )}
    >
      <Icon className={cn('size-3.5', busy && 'animate-spin')} />
    </button>
  )
}

function Th({ children }: { children: React.ReactNode }) {
  return <th className="text-left font-semibold px-3 py-2">{children}</th>
}
function Td({ children, className }: { children: React.ReactNode; className?: string }) {
  return <td className={cn('px-3 py-2.5 align-middle', className)}>{children}</td>
}

function Empty({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="h-full flex flex-col items-center justify-center text-center px-6 select-none">
      <Orbit className="h-12 w-12 text-[var(--orbit-accent)]/80" />
      <h2 className="mt-4 text-[15px] font-semibold">No projects yet</h2>
      <p className="mt-1 text-xs text-[var(--orbit-muted)] max-w-xs">
        Point Orbit at a folder with a <code className="font-mono">package.json</code>. It will
        detect the framework, package manager, and dev command automatically.
      </p>
      <Button className="mt-5" onClick={onAdd}>
        <Plus />
        Add project
      </Button>
    </div>
  )
}
