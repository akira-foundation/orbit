import { useMemo, useState } from "react";
import type { MetricSample, Project } from "../../types";
import { cn } from "../../lib/cn";

function FilterAction({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="text-[10px] text-[var(--orbit-muted)] hover:text-[var(--orbit-text)] underline-offset-2 hover:underline"
    >
      {label}
    </button>
  );
}

export function ProjectFilter({
  projects,
  visibleIds,
  colorFor,
  samples,
  onToggle,
  onReset,
  onAll,
  onNone,
}: {
  projects: Project[];
  visibleIds: string[];
  colorFor: Map<string, string>;
  samples: Record<string, MetricSample[]>;
  onToggle: (id: string) => void;
  onReset: () => void;
  onAll: () => void;
  onNone: () => void;
}) {
  const [query, setQuery] = useState("");
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return projects;
    return projects.filter((p) => p.name.toLowerCase().includes(q));
  }, [projects, query]);

  const lastConn = (id: string) =>
    samples[id]?.[samples[id].length - 1]?.conns ?? 0;

  return (
    <section className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-4 space-y-3">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-[12px] font-semibold">Projects shown</p>
          <p className="text-[10px] text-[var(--orbit-muted)]">
            {visibleIds.length} of {projects.length} · click to toggle
          </p>
        </div>
        <div className="flex items-center gap-2">
          {projects.length > 12 && (
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Filter…"
              className="h-7 px-2.5 rounded-md bg-white/5 border border-white/10 text-[11px] outline-none focus:border-white/20 placeholder:text-[var(--orbit-subtle)]"
            />
          )}
          <FilterAction label="All" onClick={onAll} />
          <FilterAction label="None" onClick={onNone} />
          <FilterAction label="Reset" onClick={onReset} />
        </div>
      </div>
      <div className="flex flex-wrap gap-1.5 max-h-32 overflow-auto scrollbar-thin">
        {filtered.map((p) => {
          const active = visibleIds.includes(p.id);
          const color = colorFor.get(p.id) ?? "#71717a";
          return (
            <button
              key={p.id}
              onClick={() => onToggle(p.id)}
              className={cn(
                "inline-flex items-center gap-1.5 h-6 px-2.5 rounded-full text-[10px] font-medium border transition-colors",
                active
                  ? "border-white/15 bg-white/10 text-white"
                  : "border-white/8 bg-transparent text-[var(--orbit-muted)] hover:text-white",
              )}
              title={
                lastConn(p.id) > 0
                  ? `${lastConn(p.id)} active connection${lastConn(p.id) > 1 ? "s" : ""}`
                  : "no recent activity"
              }
            >
              <span
                className="size-1.5 rounded-full"
                style={{ background: active ? color : "rgba(255,255,255,0.2)" }}
              />
              {p.name}
            </button>
          );
        })}
        {filtered.length === 0 && (
          <p className="text-[11px] text-[var(--orbit-muted)] italic">
            No projects match.
          </p>
        )}
      </div>
    </section>
  );
}
