import { ArrowLeft, ArrowRight, Search } from "lucide-react"
import { useProjects } from "../store"
import { Button } from "./ui/button"

export function Toolbar({ title, onOpenPalette }: { title?: string; onOpenPalette: () => void }) {
  const { selectedId, select } = useProjects()

  return (
    <div className="relative flex items-center h-full px-4">
      <div className="no-drag flex items-center gap-1">
        <Button
          variant="ghost"
          size="icon"
          disabled={!selectedId}
          onClick={() => select(null)}
        >
          <ArrowLeft />
        </Button>
        <Button variant="ghost" size="icon" disabled>
          <ArrowRight />
        </Button>
        <div className="ml-2 text-[14px] font-semibold text-[var(--orbit-text)] truncate">
          {title}
        </div>
      </div>

      <div className="absolute left-1/2 -translate-x-1/2 no-drag w-[420px] max-w-[60%]">
        <SearchTrigger onClick={onOpenPalette} />
      </div>
    </div>
  )
}

function SearchTrigger({ onClick }: { onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="group w-full flex items-center h-9 rounded-full bg-white/[0.05] border border-white/[0.08] hover:bg-white/[0.08] hover:border-white/[0.12] transition-colors text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <Search className="ml-3 h-[14px] w-[14px] text-[var(--orbit-muted)]" />
      <span className="flex-1 px-2 text-[13px] text-[var(--orbit-muted)]">Search...</span>
      <kbd className="mr-2 inline-flex h-5 min-w-[24px] items-center justify-center rounded-md px-1.5 bg-white/[0.06] border border-white/[0.10] text-[11px] text-[var(--orbit-muted)] font-medium">
        ⌘K
      </kbd>
    </button>
  )
}
