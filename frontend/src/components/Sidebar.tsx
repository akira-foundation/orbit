import { useEffect, useState } from "react";
import { useProjects, type Filter } from "../store";
import type { ProjectStatus } from "../types";
import { api } from "../api";
import { cn } from "../lib/cn";
import {
  Boxes,
  Play,
  Moon,
  Square,
  PauseCircle,
  AlertTriangle,
  Box,
  BarChart3,
  Server,
  Cpu,
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

function countFor(filter: Filter, projects: { status: ProjectStatus }[]): number {
  if (filter === "all") return projects.length;
  return projects.filter((p) => p.status === filter).length;
}

export function Sidebar({ onOpenSettings }: { onOpenSettings?: () => void }) {
  const { projects, filter, setFilter, selectedId, select, view, setView } =
    useProjects();
  const recent = projects.slice(0, 8);
  const [runningServices, setRunningServices] = useState(0);

  useEffect(() => {
    const tick = async () => {
      const list = await api.listServices();
      setRunningServices(list.filter((s) => s.status === "running").length);
    };
    tick();
    const t = setInterval(tick, 2500);
    return () => clearInterval(t);
  }, []);

  return (
    <div className="flex-1 min-h-0 flex flex-col">
      <nav className="flex-1 overflow-auto scrollbar-thin px-2 pt-2 pb-2">
        <Section title="Views">
          <Row
            icon={BarChart3}
            label="Metrics"
            active={view === "metrics"}
            onClick={() => setView("metrics")}
          />
          <Row
            icon={Server}
            label="Services"
            active={view === "services"}
            badge={runningServices > 0 ? runningServices : undefined}
            tone={runningServices > 0 ? "ok" : undefined}
            onClick={() => setView("services")}
          />
          <Row
            icon={Cpu}
            label="Runtimes"
            active={view === "runtimes"}
            onClick={() => setView("runtimes")}
          />
        </Section>

        <Section title="Projects">
          {filterItems
            .filter((it) => it.id === "all" || countFor(it.id, projects) > 0)
            .map((it) => {
              const count = countFor(it.id, projects);
              return (
                <Row
                  key={it.id}
                  icon={it.icon}
                  label={it.label}
                  badge={count > 0 ? count : undefined}
                  tone={
                    it.id === "running"
                      ? "ok"
                      : it.id === "error"
                        ? "warn"
                        : undefined
                  }
                  active={view === "projects" && !selectedId && filter === it.id}
                  onClick={() => {
                    select(null);
                    setView("projects");
                    setFilter(it.id);
                  }}
                />
              );
            })}
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

      </nav>

      <ul className="shrink-0 px-2 pb-2 space-y-1">
        <li>
          <button
            onClick={onOpenSettings}
            className="no-drag w-full h-[34px] px-3 rounded-md flex items-center gap-3 text-[13px] text-[var(--orbit-text)]/90 hover:text-[var(--orbit-text)] transition"
          >
            <Settings className="h-[17px] w-[17px] shrink-0 text-[var(--orbit-muted)]" />
            <span className="truncate flex-1 text-left">Settings</span>
          </button>
        </li>
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
  badge,
  tone,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  active?: boolean;
  muted?: boolean;
  dot?: string;
  badge?: number;
  tone?: "ok" | "warn";
  onClick?: () => void;
}) {
  const badgeCls = cn(
    "ml-auto inline-flex items-center justify-center size-5 shrink-0 rounded-full text-[10px] font-semibold tabular-nums leading-none",
    tone === "ok"
      ? "bg-emerald-400/15 text-emerald-300"
      : tone === "warn"
        ? "bg-rose-400/15 text-rose-300"
        : "bg-white/8 text-[var(--orbit-muted)]",
  );
  const badgeText =
    badge !== undefined ? (badge > 99 ? "99+" : String(badge)) : "";
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
        {badge !== undefined && <span className={badgeCls}>{badgeText}</span>}
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
