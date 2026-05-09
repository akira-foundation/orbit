import { useEffect, useMemo, useState } from 'react'
import { useProjects } from '../store'
import { ProjectIcon } from '../components/ProjectIcon'
import { Orbit, Plus } from 'lucide-react'
import { Button } from '../components/ui/button'
import type { Project } from '../types'
import { cn } from '../lib/cn'
import { RuntimeStatusBadge } from '../components/RuntimeStatusBadge'

export function Dashboard({
  view,
  onAdd,
}: {
  view: 'grid' | 'list'
  onAdd: () => void
}) {
  const { projects, filter, query, select } = useProjects()
  const [focusId, setFocusId] = useState<string | null>(null)

  // Clear local focus when filter/query changes so stale selection isn't shown
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

  const open = (p: Project) => {
    setFocusId(null)
    select(p.id)
  }

  return (
    <div className="h-full overflow-auto scrollbar-thin" onClick={() => setFocusId(null)}>
      {view === 'grid' ? (
        <GridView projects={filtered} focusId={focusId} setFocusId={setFocusId} onOpen={open} />
      ) : (
        <ListView projects={filtered} focusId={focusId} setFocusId={setFocusId} onOpen={open} />
      )}
    </div>
  )
}

// ─── Grid ────────────────────────────────────────────────────────────────────

function GridView({
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
    <div className="p-8">
      <div className="flex flex-wrap gap-x-4 gap-y-6">
        {projects.map((p) => (
          <div key={p.id} onClick={(e) => e.stopPropagation()}>
            <ProjectIcon
              project={p}
              selected={focusId === p.id}
              onSelect={() => setFocusId(p.id)}
              onOpen={() => onOpen(p)}
            />
          </div>
        ))}
      </div>
    </div>
  )
}

// ─── List ────────────────────────────────────────────────────────────────────

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
          <tr className="text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)]">
            <Th>Name</Th>
            <Th>Status</Th>
            <Th>Framework</Th>
            <Th>Manager</Th>
            <Th>Domain</Th>
          </tr>
        </thead>
        <tbody>
          {projects.map((p) => (
            <tr
              key={p.id}
              onClick={(e) => { e.stopPropagation(); setFocusId(p.id) }}
              onDoubleClick={() => onOpen(p)}
              className={cn(
                'cursor-default border-t border-[var(--orbit-border)] transition-colors',
                focusId === p.id
                  ? 'bg-[var(--orbit-accent)]/20'
                  : 'hover:bg-white/[0.04]',
              )}
            >
              <Td className="font-medium">{p.name}</Td>
              <Td><RuntimeStatusBadge status={p.status} /></Td>
              <Td>{p.detectedFramework}</Td>
              <Td>{p.packageManager}</Td>
              <Td className="font-mono text-xs text-[var(--orbit-muted)]">{p.localDomain}</Td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function Th({ children }: { children: React.ReactNode }) {
  return <th className="text-left font-semibold px-3 py-2">{children}</th>
}
function Td({ children, className }: { children: React.ReactNode; className?: string }) {
  return <td className={cn('px-3 py-2 align-middle', className)}>{children}</td>
}

// ─── Empty state ─────────────────────────────────────────────────────────────

function Empty({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="h-full flex flex-col items-center justify-center text-center px-6 select-none">
      <Orbit className="h-12 w-12 text-[var(--orbit-accent)]/80" />
      <h2 className="mt-4 text-[15px] font-semibold">No projects yet</h2>
      <p className="mt-1 text-[12px] text-[var(--orbit-muted)] max-w-xs">
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
