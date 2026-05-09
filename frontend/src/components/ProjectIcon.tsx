import type { Project } from '../types'
import { cn } from '../lib/cn'

const tints: Record<string, string> = {
  nextjs: 'from-zinc-200 to-zinc-500',
  nuxt:   'from-emerald-300 to-emerald-600',
  astro:  'from-orange-300 to-pink-500',
  vite:   'from-violet-300 to-indigo-500',
  nestjs: 'from-rose-300 to-rose-600',
  express:'from-amber-300 to-amber-600',
  unknown:'from-[var(--orbit-accent)] to-[var(--orbit-accent-2)]',
}

const dotColors: Record<string, string> = {
  running: 'bg-emerald-400',
  starting: 'bg-amber-400',
  idle: 'bg-sky-400',
  suspended: 'bg-violet-400',
  error: 'bg-rose-500',
  stopped: 'bg-zinc-400',
}

export function ProjectIcon({ project, selected, onSelect, onOpen }: {
  project: Project
  selected: boolean
  onSelect: () => void
  onOpen: () => void
}) {
  const tint = tints[project.detectedFramework] ?? tints.unknown

  return (
    <button
      onClick={onSelect}
      onDoubleClick={onOpen}
      className="group flex flex-col items-center gap-2 w-28 outline-none"
    >
      <div className={cn(
        'relative h-20 w-20 rounded-2xl border transition',
        'bg-gradient-to-br', tint,
        'border-white/20',
        'shadow-[inset_0_1px_0_rgba(255,255,255,0.35),0_8px_24px_rgba(0,0,0,0.35)]',
        selected && 'ring-2 ring-[var(--orbit-accent)]/70 ring-offset-2 ring-offset-transparent',
      )}>
        <span className="absolute inset-0 rounded-2xl bg-gradient-to-b from-white/25 to-transparent opacity-60" />
        <span className="absolute inset-x-3 top-2 h-2 rounded-full bg-white/40 blur-[2px]" />
        <span className="absolute bottom-2 right-2 flex items-center gap-1 px-1.5 h-4 rounded-full bg-black/40 text-[9px] uppercase tracking-wider text-white/90 backdrop-blur-md">
          <span className={cn('h-1 w-1 rounded-full', dotColors[project.status])} />
          {project.status}
        </span>
      </div>
      <span className={cn(
        'text-xs px-1.5 py-0.5 rounded-md max-w-[7rem] truncate',
        selected ? 'bg-[var(--orbit-accent)]/70 text-white' : 'text-[var(--orbit-text)]',
      )}>
        {project.name}
      </span>
    </button>
  )
}
