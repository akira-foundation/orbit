import { useMemo } from "react";
import { AlertTriangle, Plug } from "lucide-react";
import type { MetricSample, Project } from "../types";
import { cn } from "../lib/cn";

const P95_ATTENTION_MS = 1000;

function computeAttention(
  projects: Project[],
  samples: Record<string, MetricSample[]>,
) {
  const byPort = new Map<number, Project[]>();
  projects.forEach((p) => {
    if (p.devPort > 0) {
      const list = byPort.get(p.devPort) ?? [];
      list.push(p);
      byPort.set(p.devPort, list);
    }
  });
  const portConflicts = Array.from(byPort.entries())
    .filter(([, list]) => list.length > 1)
    .map(([port, list]) => ({ port, projects: list }));

  const errored = projects.filter((p) => p.status === "error");

  const crashed: { project: Project; count: number }[] = [];
  const slow: { project: Project; p95: number }[] = [];
  projects.forEach((p) => {
    const buf = samples[p.id];
    if (!buf || buf.length === 0) return;
    const crashes = buf.reduce((n, s) => n + (s.crashes ?? 0), 0);
    if (crashes > 0) crashed.push({ project: p, count: crashes });
    const last = buf[buf.length - 1];
    if (last.p95Ms >= P95_ATTENTION_MS) slow.push({ project: p, p95: last.p95Ms });
  });

  const count =
    portConflicts.length + errored.length + crashed.length + slow.length;
  return { portConflicts, errored, crashed, slow, count };
}

export function AttentionStrip({
  projects,
  samples,
  onSelect,
}: {
  projects: Project[];
  samples: Record<string, MetricSample[]>;
  onSelect: (id: string) => void;
}) {
  const a = useMemo(() => computeAttention(projects, samples), [projects, samples]);
  if (a.count === 0) return null;

  return (
    <section className="rounded-2xl border border-amber-400/20 bg-amber-400/[0.04] p-4 space-y-2.5">
      <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-widest text-amber-300/90">
        <AlertTriangle className="size-3" />
        Needs attention · {a.count}
      </div>
      <div className="flex flex-wrap gap-1.5">
        {a.portConflicts.map((c) => (
          <span
            key={`port-${c.port}`}
            className="inline-flex items-center gap-1.5 h-6 px-2.5 rounded-full text-[10px] font-medium border border-amber-400/20 bg-amber-400/5 text-amber-200"
            title={c.projects.map((p) => p.name).join(", ")}
          >
            <Plug className="size-3" />
            Port {c.port}: {c.projects.map((p) => p.name).join(" · ")}
          </span>
        ))}
        {a.errored.map((p) => (
          <AttentionChip
            key={`err-${p.id}`}
            tone="bad"
            label={`${p.name} · error`}
            onClick={() => onSelect(p.id)}
          />
        ))}
        {a.crashed.map(({ project, count }) => (
          <AttentionChip
            key={`crash-${project.id}`}
            tone="bad"
            label={`${project.name} · ${count} crash${count > 1 ? "es" : ""}`}
            onClick={() => onSelect(project.id)}
          />
        ))}
        {a.slow.map(({ project, p95 }) => (
          <AttentionChip
            key={`slow-${project.id}`}
            tone="warn"
            label={`${project.name} · P95 ${p95}ms`}
            onClick={() => onSelect(project.id)}
          />
        ))}
      </div>
    </section>
  );
}

function AttentionChip({
  label,
  tone,
  onClick,
}: {
  label: string;
  tone: "warn" | "bad";
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "inline-flex items-center gap-1.5 h-6 px-2.5 rounded-full text-[10px] font-medium border transition-colors",
        tone === "bad"
          ? "border-rose-400/25 bg-rose-400/5 text-rose-200 hover:bg-rose-400/10"
          : "border-amber-400/25 bg-amber-400/5 text-amber-200 hover:bg-amber-400/10",
      )}
    >
      {label}
    </button>
  );
}
