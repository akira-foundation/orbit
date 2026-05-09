import { useEffect, useState } from 'react'
import { useProjects } from '../store'
import { ProjectCard } from '../components/ProjectCard'
import { Button } from '../components/Button'
import { AddProjectDialog } from '../components/AddProjectDialog'
import { Plus, Orbit } from 'lucide-react'

export function Dashboard() {
  const { projects, loading, error, load, add, select } = useProjects()
  const [dialog, setDialog] = useState(false)

  useEffect(() => { load() }, [load])

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Projects</h1>
          <p className="text-sm text-[var(--color-muted)]">Add a project once. Open its URL — Orbit handles the rest.</p>
        </div>
        <Button variant="primary" onClick={() => setDialog(true)}><Plus className="h-4 w-4" />Add project</Button>
      </div>

      {error && <div className="rounded-lg bg-rose-500/10 ring-1 ring-rose-500/30 text-rose-300 text-sm px-3 py-2">{error}</div>}

      {!loading && projects.length === 0 && (
        <EmptyState onAdd={() => setDialog(true)} />
      )}

      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
        {projects.map(p => (
          <ProjectCard key={p.id} project={p} onOpen={(p) => select(p.id)} />
        ))}
      </div>

      <AddProjectDialog
        open={dialog}
        onClose={() => setDialog(false)}
        onCreated={(p) => add(p)}
      />
    </div>
  )
}

function EmptyState({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="rounded-2xl bg-[var(--color-surface)] ring-1 ring-[var(--color-border)] p-10 text-center">
      <Orbit className="h-10 w-10 mx-auto text-[var(--color-accent)]" />
      <h2 className="mt-4 text-lg font-semibold">No projects yet</h2>
      <p className="mt-1 text-sm text-[var(--color-muted)]">Point Orbit to a folder. It detects the framework, package manager, and dev command.</p>
      <div className="mt-5"><Button variant="primary" onClick={onAdd}><Plus className="h-4 w-4" />Add your first project</Button></div>
    </div>
  )
}
