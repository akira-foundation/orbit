import { Orbit } from "lucide-react"

export function Logo({ size = 28 }: { size?: number }) {
  return (
    <div className="flex items-center gap-2">
      <Orbit
        width={size}
        height={size}
        strokeWidth={2}
        className="text-[var(--orbit-accent)]"
      />
      <div className="leading-tight">
        <div className="text-sm font-semibold tracking-tight">Orbit</div>
        <div className="text-[10px] uppercase tracking-[0.18em] text-[var(--orbit-muted)]">Smart Runtime Orchestration</div>
      </div>
    </div>
  )
}
