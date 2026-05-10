import { useProjects, type Filter } from "../store";
import type { ProjectStatus } from "../types";
import { cn } from "../lib/cn";
import {
  Boxes,
  Play,
  Moon,
  Square,
  PauseCircle,
  AlertTriangle,
  Folder,
  Star,
  Clock,
  Box,
  Settings,
} from "lucide-react";

interface Item {
  id: Filter;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}

const filterItems: Item[] = [
  { id: "all", label: "All Projects", icon: Boxes },
  { id: "running", label: "Running", icon: Play },
  { id: "idle", label: "Idle", icon: Moon },
  { id: "stopped", label: "Stopped", icon: Square },
  { id: "suspended", label: "Suspended", icon: PauseCircle },
  { id: "error", label: "Errors", icon: AlertTriangle },
];

export function Sidebar({ onOpenSettings }: { onOpenSettings?: () => void }) {
  const { projects, filter, setFilter, selectedId, select } = useProjects();
  const recent = projects.slice(0, 8);

  return (
    <div className="flex-1 min-h-0 flex flex-col">
      <nav className="flex-1 overflow-auto scrollbar-thin px-2 pt-2 pb-2">
        <Section title="Workspace">
          {filterItems.map((it) => (
            <Row
              key={it.id}
              icon={it.icon}
              label={it.label}
              active={!selectedId && filter === it.id}
              onClick={() => {
                select(null);
                setFilter(it.id);
              }}
            />
          ))}
        </Section>

        {recent.length > 0 && (
          <Section title="Recent">
            {recent.map((p) => (
              <Row
                key={p.id}
                icon={Box}
                label={p.name}
                dot={dotForStatus(p.status)}
                active={selectedId === p.id}
                onClick={() => select(p.id)}
              />
            ))}
          </Section>
        )}

        <Section title="Locations">
          <Row icon={Folder} label="Local Projects" muted />
          <Row icon={Star} label="Favorites" muted />
          <Row icon={Clock} label="Last 7 days" muted />
        </Section>
      </nav>

      <ul className="shrink-0 px-2 pb-2 space-y-1">
        <Row icon={Settings} label="Settings" onClick={onOpenSettings} />
      </ul>
    </div>
  );
}

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="mt-5 first:mt-3">
      <h3 className="px-3 mb-1.5 text-[11px] font-semibold text-[var(--orbit-subtle)]">
        {title}
      </h3>
      <ul className="space-y-1">{children}</ul>
    </div>
  );
}

function Row({
  icon: Icon,
  label,
  active,
  onClick,
  muted,
  dot,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  active?: boolean;
  muted?: boolean;
  dot?: string;
  onClick?: () => void;
}) {
  return (
    <li>
      <button
        onClick={onClick}
        className={cn(
          "no-drag w-full h-[34px] px-3 rounded-md flex items-center gap-3 text-[13px] transition",
          active
            ? "bg-white/[0.10] text-[var(--orbit-text)]"
            : "text-[var(--orbit-text)]/90 hover:bg-white/[0.06]",
          muted && "cursor-default hover:bg-transparent",
        )}
      >
        <Icon className="h-[17px] w-[17px] shrink-0 text-[var(--orbit-muted)]" />
        <span className="truncate flex-1 text-left">{label}</span>
        {dot && <span className={cn("h-1.5 w-1.5 rounded-full", dot)} />}
      </button>
    </li>
  );
}

function dotForStatus(s: ProjectStatus): string {
  switch (s) {
    case "running":
      return "bg-emerald-400";
    case "starting":
      return "bg-amber-400";
    case "idle":
      return "bg-sky-400";
    case "suspended":
      return "bg-violet-400";
    case "error":
      return "bg-rose-500";
    default:
      return "bg-zinc-400";
  }
}
