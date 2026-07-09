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
  BarChart3, Server, Cpu, Mail, Database, HardDrive, Settings,
} from "lucide-react"
import type { View } from "../store"

interface FilterEntry {
  id: Filter
  label: string
  icon: React.ComponentType<{ className?: string }>
  shortcut?: string
}

const views: {
  id: View
  label: string
  icon: React.ComponentType<{ className?: string }>
}[] = [
  { id: "mail",     label: "Mail",     icon: Mail },
  { id: "database", label: "Database", icon: Database },
  { id: "storage",  label: "Storage",  icon: HardDrive },
  { id: "metrics",  label: "Metrics",  icon: BarChart3 },
  { id: "services", label: "Services", icon: Server },
  { id: "runtimes", label: "Runtimes", icon: Cpu },
  { id: "settings", label: "Settings", icon: Settings },
]

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
  const { projects, select, setFilter, setView } = useProjects()

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

              <CommandGroup heading="Go to">
                {views.map((v) => {
                  const Icon = v.icon
                  return (
                    <CommandItem
                      key={v.id}
                      value={`go to ${v.label}`}
                      onSelect={() => {
                        select(null)
                        setView(v.id)
                        close()
                      }}
                    >
                      <Icon />
                      {v.label}
                    </CommandItem>
                  )
                })}
              </CommandGroup>

              <CommandGroup heading="Filters">
                {filters.map((f) => {
                  const Icon = f.icon
                  return (
                    <CommandItem
                      key={f.id}
                      value={`filter ${f.label}`}
                      onSelect={() => {
                        select(null)
                        setView("projects")
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
