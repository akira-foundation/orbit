export function Logo({ size = 28 }: { size?: number }) {
  return (
    <div className="flex items-center gap-2">
      <svg width={size} height={size} viewBox="0 0 32 32" fill="none">
        <defs>
          <linearGradient id="og" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
            <stop stopColor="#7c5cff" />
            <stop offset="1" stopColor="#5cc8ff" />
          </linearGradient>
        </defs>
        <circle cx="16" cy="16" r="6" fill="url(#og)" />
        <ellipse cx="16" cy="16" rx="13" ry="5" stroke="url(#og)" strokeWidth="1.5" transform="rotate(-25 16 16)" />
      </svg>
      <div className="leading-tight">
        <div className="text-sm font-semibold tracking-tight">Orbit</div>
        <div className="text-[10px] uppercase tracking-[0.18em] text-[var(--orbit-muted)]">Smart Runtime Orchestration</div>
      </div>
    </div>
  )
}
