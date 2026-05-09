import type { ProjectStatus } from '../types'
import { cn } from '../lib/cn'

const map: Record<ProjectStatus, { label: string; dot: string; text: string; ring: string }> = {
  stopped:   { label: 'Stopped',   dot: 'bg-zinc-400',    text: 'text-zinc-200',    ring: 'ring-white/10' },
  starting:  { label: 'Starting',  dot: 'bg-amber-400 animate-pulse', text: 'text-amber-200', ring: 'ring-amber-400/30' },
  running:   { label: 'Running',   dot: 'bg-emerald-400', text: 'text-emerald-200', ring: 'ring-emerald-400/30' },
  idle:      { label: 'Idle',      dot: 'bg-sky-400',     text: 'text-sky-200',     ring: 'ring-sky-400/30' },
  suspended: { label: 'Suspended', dot: 'bg-violet-400',  text: 'text-violet-200',  ring: 'ring-violet-400/30' },
  error:     { label: 'Error',     dot: 'bg-rose-500',    text: 'text-rose-200',    ring: 'ring-rose-400/30' },
}

export function RuntimeStatusBadge({ status }: { status: ProjectStatus }) {
  const s = map[status] ?? map.stopped
  return (
    <span className={cn(
      'inline-flex items-center gap-2 rounded-full px-2.5 py-1 text-xs ring-1 backdrop-blur-md',
      s.text, s.ring, 'bg-white/[0.06]',
    )}>
      <span className={cn('h-1.5 w-1.5 rounded-full', s.dot)} />
      {s.label}
    </span>
  )
}
