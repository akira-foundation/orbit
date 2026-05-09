import {
  ArrowLeft,
  ArrowRight,
  LayoutGrid,
  List,
  Plus,
  Search,
} from "lucide-react"
import { useProjects } from "../store"
import { cn } from "../lib/cn"
import { Button } from "./ui/button"
import { Input } from "./ui/input"

export function Toolbar({
  onAdd,
  view,
  onViewChange,
  title,
}: {
  onAdd: () => void
  view: "grid" | "list"
  onViewChange: (v: "grid" | "list") => void
  title?: string
}) {
  const { query, setQuery, selectedId, select } = useProjects()

  return (
    <div className="flex items-center gap-2 h-full px-4">
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
      </div>

      <div className="ml-2 text-[14px] font-semibold text-[var(--orbit-text)] truncate">
        {title}
      </div>

      <div className="flex-1" />

      <div className="no-drag flex items-center p-0.5 rounded-md bg-white/[0.06] border border-white/[0.10]">
        <SegBtn active={view === "grid"} onClick={() => onViewChange("grid")}>
          <LayoutGrid className="h-[14px] w-[14px]" />
        </SegBtn>
        <SegBtn active={view === "list"} onClick={() => onViewChange("list")}>
          <List className="h-[14px] w-[14px]" />
        </SegBtn>
      </div>

      <Button size="sm" onClick={onAdd} className="no-drag">
        <Plus />
        Add
      </Button>

      <div className="no-drag relative">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-[13px] w-[13px] text-[var(--orbit-muted)] pointer-events-none" />
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search"
          className="h-7 w-44 pl-7 text-[12px]"
        />
      </div>
    </div>
  )
}

function SegBtn({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "h-6 w-7 rounded flex items-center justify-center transition",
        active
          ? "bg-white/[0.16] text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.2)]"
          : "text-[var(--orbit-muted)] hover:text-white",
      )}
    >
      {children}
    </button>
  )
}
