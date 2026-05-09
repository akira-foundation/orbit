import { useProjects } from '../store'

export function StatusBar({ count }: { count: number }) {
  const total = useProjects(s => s.projects.length)
  return (
    <div className="h-6 px-3 flex items-center justify-center text-[11px] text-[var(--orbit-muted)]">
      {count === total
        ? `${total} ${total === 1 ? 'project' : 'projects'}`
        : `${count} of ${total} shown`}
    </div>
  )
}
