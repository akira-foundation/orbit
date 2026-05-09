import type { ProjectStatus } from '../types'
import { cn } from '../lib/cn'

const map: Record<ProjectStatus, { label: string; dot: string; text: string; ring: string }> = {
  stopped:   { label: 'Stopped',   dot: 'bg-zinc-500',   text: 'text-zinc-400',   ring: 'ring-zinc-500/20' },
  starting:  { label: 'Starting',  dot: 'bg-amber-400 animate-pulse',  text: 'text-amber-300',  ring: 'ring-amber-500/30' },
  running:   { label: 'Running',   dot: 'bg-emerald-400', text: 'text-emerald-300', ring: 'ring-emerald-500/30' },
  idle:      { label: 'Idle',      dot: 'bg-sky-400',     text: 'text-sky-300',     ring: 'ring-sky-500/30' },
  suspended: { label: 'Suspended', dot: 'bg-violet-400',  text: 'text-violet-300',  ring: 'ring-violet-500/30' },
  error:     { label: 'Error',     dot: 'bg-rose-500',    text: 'text-rose-300',    ring: 'ring-rose-500/30' },
}

export function RuntimeStatusBadge({ status }: { status: ProjectStatus }) {
  const s = map[status] ?? map.stopped
  return (
    <span className={cn('inline-flex items-center gap-2 rounded-full px-2.5 py-1 text-xs ring-1', s.text, s.ring, 'bg-[var(--color-surface-2)]')}>
      <span className={cn('h-1.5 w-1.5 rounded-full', s.dot)} />
      {s.label}
    </span>
  )
}
