import { useEffect } from "react"
import { useProjects, type Filter } from "../store"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandShortcut,
} from "./ui/command"
import {
  Dialog,
  DialogContent,
  DialogPortal,
  DialogOverlay,
} from "./ui/dialog"
import {
  Boxes, Play, Moon, Square, PauseCircle, AlertTriangle, Box, Plus,
} from "lucide-react"

interface FilterEntry {
  id: Filter
  label: string
  icon: React.ComponentType<{ className?: string }>
  shortcut?: string
}

const filters: FilterEntry[] = [
  { id: "all",       label: "All Projects", icon: Boxes,         shortcut: "⌘1" },
  { id: "running",   label: "Running",      icon: Play,          shortcut: "⌘2" },
  { id: "idle",      label: "Idle",         icon: Moon,          shortcut: "⌘3" },
  { id: "stopped",   label: "Stopped",      icon: Square,        shortcut: "⌘4" },
  { id: "suspended", label: "Suspended",    icon: PauseCircle,   shortcut: "⌘5" },
  { id: "error",     label: "Errors",       icon: AlertTriangle, shortcut: "⌘6" },
]

export function CommandPalette({
  open,
  onOpenChange,
  onAdd,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
  onAdd: () => void
}) {
  const { projects, select, setFilter } = useProjects()

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault()
        onOpenChange(!open)
      }
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [open, onOpenChange])

  const close = () => onOpenChange(false)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogPortal>
        <DialogOverlay />
        <DialogContent className="p-0 max-w-[640px] top-[20%] translate-y-0">
          <Command loop>
            <CommandInput placeholder="Search projects or commands..." />
            <CommandList>
              <CommandEmpty>No results.</CommandEmpty>

              {projects.length > 0 && (
                <CommandGroup heading="Projects">
                  {projects.map((p) => (
                    <CommandItem
                      key={p.id}
                      value={`project ${p.name} ${p.detectedFramework} ${p.localDomain}`}
                      onSelect={() => {
                        select(p.id)
                        close()
                      }}
                    >
                      <Box />
                      <span className="truncate">{p.name}</span>
                      <span className="ml-auto text-[11px] text-[var(--orbit-muted)] truncate max-w-[160px]">
                        {p.localDomain}
                      </span>
                    </CommandItem>
                  ))}
                </CommandGroup>
              )}

              <CommandGroup heading="Filters">
                {filters.map((f) => {
                  const Icon = f.icon
                  return (
                    <CommandItem
                      key={f.id}
                      value={`filter ${f.label}`}
                      onSelect={() => {
                        select(null)
                        setFilter(f.id)
                        close()
                      }}
                    >
                      <Icon />
                      {f.label}
                      {f.shortcut && <CommandShortcut>{f.shortcut}</CommandShortcut>}
                    </CommandItem>
                  )
                })}
              </CommandGroup>

              <CommandGroup heading="Actions">
                <CommandItem
                  value="action add project"
                  onSelect={() => {
                    close()
                    onAdd()
                  }}
                >
                  <Plus />
                  Add Project
                  <CommandShortcut>⌘N</CommandShortcut>
                </CommandItem>
              </CommandGroup>
            </CommandList>
          </Command>
        </DialogContent>
      </DialogPortal>
    </Dialog>
  )
}
