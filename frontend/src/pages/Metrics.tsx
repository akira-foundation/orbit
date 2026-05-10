import { useEffect, useMemo, useState } from "react";
import { useInterval, useLocalStorage } from "usehooks-ts";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import {
  Activity,
  AlertTriangle,
  ArrowUpRight,
  Cpu,
  Gauge,
  Plug,
  RotateCw,
  Zap,
} from "lucide-react";
import { api } from "../api";
import { useProjects } from "../store";
import type { MetricSample, Project } from "../types";
import { cn } from "../lib/cn";
import { MetricsFilter, MetricInterval, getSinceTsForInterval } from "../components/MetricsFilter";
import { Skeleton } from "../components/ui/skeleton";

// Lightweight palette so each project line has a stable color.
const PALETTE = [
  "#22d3ee",
  "#a78bfa",
  "#34d399",
  "#f472b6",
  "#fbbf24",
  "#60a5fa",
  "#fb7185",
  "#4ade80",
];

const MAX_DEFAULT_VISIBLE = 10;

export function MetricsPage() {
  const { projects, showProjectMetrics } = useProjects();
  const [samples, setSamples] = useState<Record<string, MetricSample[]>>({});
  const [loaded, setLoaded] = useState(false);
  const [selected, setSelected] = useLocalStorage<string[] | null>(
    "orbit:metrics:selectedProjects",
    null,
  );

  const [range, setRange] = useLocalStorage<MetricInterval>("orbit:metrics:interval", "1H");

  const visibleIds = useMemo(() => {
    if (selected !== null) {
      const valid = new Set(projects.map((p) => p.id));
      return selected.filter((id) => valid.has(id));
    }
    // Default: most recently created N projects. Sample-based ranking
    // would create a chicken-and-egg with the visibleIds-filtered fetch.
    return [...projects]
      .sort((a, b) => (b.createdAt > a.createdAt ? 1 : -1))
      .slice(0, MAX_DEFAULT_VISIBLE)
      .map((p) => p.id);
  }, [projects, selected]);

  const refresh = async () => {
    try {
      setSamples(
        await api.runtimeMetricsAll(getSinceTsForInterval(range), visibleIds),
      );
    } catch {
      /* ignore */
    } finally {
      setLoaded(true);
    }
  };
  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [range, visibleIds.join(",")]);
  useInterval(refresh, 30000);

  const visibleProjects = useMemo(
    () => projects.filter((p) => visibleIds.includes(p.id)),
    [projects, visibleIds],
  );

  const colorFor = useMemo(() => {
    const m = new Map<string, string>();
    projects.forEach((p, i) => m.set(p.id, PALETTE[i % PALETTE.length]));
    return m;
  }, [projects]);

  const toggle = (id: string) => {
    const current = visibleIds;
    const next = current.includes(id)
      ? current.filter((x) => x !== id)
      : [...current, id];
    setSelected(next);
  };

  const totals = useMemo(() => computeTotals(visibleProjects, samples), [
    visibleProjects,
    samples,
  ]);
  const merged = useMemo(() => mergeForChart(visibleProjects, samples), [
    visibleProjects,
    samples,
  ]);
  const memMerged = useMemo(() => mergeMemChart(visibleProjects, samples), [
    visibleProjects,
    samples,
  ]);

  if (!loaded) {
    return (
      <div className="h-full overflow-auto scrollbar-thin">
        <div className="mx-auto w-full max-w-6xl px-10 py-8 space-y-6">
          <header className="flex items-center justify-between">
            <div className="space-y-2">
              <Skeleton className="h-7 w-24" />
              <Skeleton className="h-3 w-32" />
            </div>
            <Skeleton className="h-8 w-56 rounded-lg" />
          </header>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Skeleton className="h-36 rounded-2xl" />
            <Skeleton className="h-36 rounded-2xl" />
            <Skeleton className="h-36 rounded-2xl" />
          </div>
          <Skeleton className="h-44 rounded-2xl" />
          <Skeleton className="h-32 rounded-2xl" />
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <Skeleton className="h-64 rounded-2xl" />
            <Skeleton className="h-64 rounded-2xl" />
            <Skeleton className="h-64 rounded-2xl" />
            <Skeleton className="h-64 rounded-2xl" />
          </div>
          <Skeleton className="h-48 rounded-2xl" />
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-auto scrollbar-thin">
      <div className="mx-auto w-full max-w-6xl px-10 py-8 space-y-6">
        <header className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">Metrics</h1>
            <p className="text-xs text-[var(--orbit-muted)] mt-1">
              5s sampling · live
            </p>
          </div>
          <MetricsFilter value={range} onChange={setRange} />
        </header>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <AggregateStatusCard visible={visibleProjects} samples={samples} />
          <AggregateLatencyCard visible={visibleProjects} samples={samples} />
          <AggregateVolumeCard visible={visibleProjects} samples={samples} />
        </div>

        <FleetSummaryStrip
          projects={projects}
          totals={totals}
          samples={samples}
          visibleProjects={visibleProjects}
        />

        <ProjectFilter
          projects={projects}
          visibleIds={visibleIds}
          colorFor={colorFor}
          samples={samples}
          onToggle={toggle}
          onReset={() => setSelected(null)}
        />

        {/* Time-series charts in a 2-column grid so the page never goes
            full-bleed on wide viewports. Heatmap stays alone because it
            needs the wider strip to render 24 hours per row. */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <ChartCard
            title="Active connections"
            subtitle="Open HTTP / WebSocket connections per project"
          >
            <ResponsiveContainer width="100%" height={220}>
              <AreaChart data={merged} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
                <defs>
                  {visibleProjects.map((p) => (
                    <linearGradient
                      key={p.id}
                      id={`g-${p.id}`}
                      x1="0"
                      y1="0"
                      x2="0"
                      y2="1"
                    >
                      <stop offset="0%" stopColor={colorFor.get(p.id)} stopOpacity={0.45} />
                      <stop offset="100%" stopColor={colorFor.get(p.id)} stopOpacity={0} />
                    </linearGradient>
                  ))}
                </defs>
                <CartesianGrid stroke="rgba(255,255,255,0.05)" vertical={false} />
                <XAxis dataKey="ts" tickFormatter={tickTime} stroke="#52525b" fontSize={10} />
                <YAxis stroke="#52525b" fontSize={10} allowDecimals={false} />
                <Tooltip
                  contentStyle={tooltipStyle}
                  labelFormatter={(v) => new Date(Number(v) * 1000).toLocaleTimeString()}
                />
                {visibleProjects.map((p) => (
                  <Area
                    key={p.id}
                    type="monotone"
                    dataKey={`${p.id}.conns`}
                    name={p.name}
                    stroke={colorFor.get(p.id)}
                    fill={`url(#g-${p.id})`}
                    strokeWidth={1.5}
                    isAnimationActive={false}
                    connectNulls
                  />
                ))}
              </AreaChart>
            </ResponsiveContainer>
          </ChartCard>

          <ChartCard title="Uptime" subtitle="Seconds since each runtime became ready">
            <ResponsiveContainer width="100%" height={220}>
              <LineChart data={merged} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
                <CartesianGrid stroke="rgba(255,255,255,0.05)" vertical={false} />
                <XAxis dataKey="ts" tickFormatter={tickTime} stroke="#52525b" fontSize={10} />
                <YAxis stroke="#52525b" fontSize={10} tickFormatter={(v) => `${Math.round(v)}s`} />
                <Tooltip
                  contentStyle={tooltipStyle}
                  labelFormatter={(v) => new Date(Number(v) * 1000).toLocaleTimeString()}
                  formatter={(v) => `${Math.round(Number(v))}s`}
                />
                {visibleProjects.map((p) => (
                  <Line
                    key={p.id}
                    type="monotone"
                    dataKey={`${p.id}.uptimeS`}
                    name={p.name}
                    stroke={colorFor.get(p.id)}
                    strokeWidth={1.5}
                    dot={false}
                    isAnimationActive={false}
                    connectNulls
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </ChartCard>

          <UsageHeatmapAggregate samples={samples} />

          <ChartCard
            title="Memory"
            subtitle="RSS per project over time"
          >
            <ResponsiveContainer width="100%" height={220}>
              <LineChart data={memMerged} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
                <CartesianGrid stroke="rgba(255,255,255,0.05)" vertical={false} />
                <XAxis dataKey="ts" tickFormatter={tickTime} stroke="#52525b" fontSize={10} />
                <YAxis
                  stroke="#52525b"
                  fontSize={10}
                  tickFormatter={(v) => formatKB(Number(v))}
                />
                <Tooltip
                  contentStyle={tooltipStyle}
                  labelFormatter={(v) => new Date(Number(v) * 1000).toLocaleTimeString()}
                  formatter={(v) => formatKB(Number(v))}
                />
                {visibleProjects.map((p) => (
                  <Line
                    key={p.id}
                    type="monotone"
                    dataKey={`${p.id}.mem`}
                    name={p.name}
                    stroke={colorFor.get(p.id)}
                    strokeWidth={1.4}
                    dot={false}
                    isAnimationActive={false}
                    connectNulls
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </ChartCard>
        </div>

        <FleetHealthTable
          projects={visibleProjects}
          samples={samples}
          onSelect={showProjectMetrics}
        />
      </div>
    </div>
  );
}

// ─── Project filter chips ────────────────────────────────────────────────

function ProjectFilter({
  projects,
  visibleIds,
  colorFor,
  samples,
  onToggle,
  onReset,
}: {
  projects: Project[];
  visibleIds: string[];
  colorFor: Map<string, string>;
  samples: Record<string, MetricSample[]>;
  onToggle: (id: string) => void;
  onReset: () => void;
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
          <button
            onClick={onReset}
            className="text-[10px] text-[var(--orbit-muted)] hover:text-[var(--orbit-text)] underline-offset-2 hover:underline"
          >
            Reset
          </button>
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

// ─── Aggregations ────────────────────────────────────────────────────────

function tickTime(v: number) {
  return new Date(v * 1000).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
}

const tooltipStyle: React.CSSProperties = {
  background: "rgba(20,21,28,0.95)",
  border: "1px solid rgba(255,255,255,0.08)",
  borderRadius: 8,
  fontSize: 11,
};

function computeTotals(
  projects: Project[],
  samples: Record<string, MetricSample[]>,
) {
  let running = 0;
  let conns = 0;
  let restarts = 0;
  let reqs = 0;
  projects.forEach((p) => {
    const buf = samples[p.id];
    const last = buf?.[buf.length - 1];
    if (!last) return;
    if (last.status === "running" || last.status === "starting") running++;
    conns += last.conns;
    restarts += last.attempts;
    reqs += last.reqCount;
  });
  return { running, conns, restarts, rps: reqs / 5 };
}

function latestAttempts(
  projectId: string,
  samples: Record<string, MetricSample[]>,
): number {
  const buf = samples[projectId];
  return buf?.[buf.length - 1]?.attempts ?? 0;
}

function mergeMemChart(
  projects: Project[],
  samples: Record<string, MetricSample[]>,
): Array<Record<string, number>> {
  const tsSet = new Set<number>();
  projects.forEach((p) =>
    (samples[p.id] ?? []).forEach((s) => tsSet.add(s.ts)),
  );
  const ts = Array.from(tsSet).sort((a, b) => a - b);
  return ts.map((t) => {
    const row: Record<string, number> = { ts: t };
    projects.forEach((p) => {
      const s = (samples[p.id] ?? []).find((x) => x.ts === t);
      row[`${p.id}.mem`] = s?.memKb ?? 0;
    });
    return row;
  });
}

function mergeForChart(
  projects: Project[],
  samples: Record<string, MetricSample[]>,
): Array<Record<string, number>> {
  const tsSet = new Set<number>();
  projects.forEach((p) => {
    (samples[p.id] ?? []).forEach((s) => tsSet.add(s.ts));
  });
  const ts = Array.from(tsSet).sort((a, b) => a - b);
  return ts.map((t) => {
    const row: Record<string, number> = { ts: t };
    projects.forEach((p) => {
      const sample = (samples[p.id] ?? []).find((s) => s.ts === t);
      if (sample) {
        row[`${p.id}.conns`] = sample.conns;
        row[`${p.id}.uptimeS`] = Math.round(sample.uptimeMs / 1000);
      }
    });
    return row;
  });
}

function aggregate(
  samples: Record<string, MetricSample[]>,
  projects: Project[],
) {
  let totalReq = 0;
  let totalErr = 0;
  const lats: number[] = [];
  projects.forEach((p) => {
    (samples[p.id] ?? []).forEach((s) => {
      totalReq += s.reqCount;
      totalErr += s.errCount;
      if (s.p95Ms > 0) lats.push(s.p95Ms);
    });
  });
  return { totalReq, totalErr, lats };
}

// ─── Aggregate cards ─────────────────────────────────────────────────────

function AggregateStatusCard({
  visible,
  samples,
}: {
  visible: Project[];
  samples: Record<string, MetricSample[]>;
}) {
  const a = aggregate(samples, visible);
  const ok = Math.max(a.totalReq - a.totalErr, 0);
  const errRate = a.totalReq > 0 ? (a.totalErr / a.totalReq) * 100 : 0;
  const data =
    a.totalReq > 0
      ? [
          { name: "2XX/3XX", value: ok, color: "#34d399" },
          { name: "4XX/5XX", value: a.totalErr, color: "#fbbf24" },
        ]
      : [{ name: "—", value: 1, color: "rgba(255,255,255,0.06)" }];
  return (
    <div className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-4">
      <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)] mb-2">
        <Activity className="size-3" />
        Response status · fleet
      </div>
      <div className="flex items-center gap-4">
        <div className="relative size-[110px] shrink-0">
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={data}
                innerRadius={36}
                outerRadius={52}
                paddingAngle={a.totalReq > 0 ? 2 : 0}
                dataKey="value"
                strokeWidth={0}
                isAnimationActive={false}
              >
                {data.map((d, i) => (
                  <Cell key={i} fill={d.color} />
                ))}
              </Pie>
            </PieChart>
          </ResponsiveContainer>
          <div className="absolute inset-0 flex flex-col items-center justify-center">
            <span className="text-xl font-mono font-semibold">{a.totalReq}</span>
            <span className="text-[9px] uppercase tracking-widest text-[var(--orbit-subtle)]">
              reqs
            </span>
          </div>
        </div>
        <div className="flex-1 min-w-0 space-y-1.5 text-[12px]">
          <div className="flex items-center justify-between gap-3">
            <span className="inline-flex items-center gap-1.5">
              <span className="size-1.5 rounded-full bg-emerald-400" />
              2xx / 3xx
            </span>
            <span className="font-mono">{ok}</span>
          </div>
          <div className="flex items-center justify-between gap-3">
            <span className="inline-flex items-center gap-1.5">
              <span className="size-1.5 rounded-full bg-amber-400" />
              4xx / 5xx
            </span>
            <span className="font-mono">{a.totalErr}</span>
          </div>
          <div className="flex items-center justify-between gap-3 pt-1.5 border-t border-white/5">
            <span className="text-[var(--orbit-muted)]">Error rate</span>
            <span
              className={cn(
                "font-mono",
                errRate > 5 ? "text-rose-300" : "text-[var(--orbit-text)]",
              )}
            >
              {errRate.toFixed(1)}%
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

function AggregateLatencyCard({
  visible,
  samples,
}: {
  visible: Project[];
  samples: Record<string, MetricSample[]>;
}) {
  const a = aggregate(samples, visible);
  const lats = a.lats.slice().sort((x, y) => x - y);
  const pick = (p: number) =>
    lats.length > 0 ? lats[Math.floor((lats.length - 1) * p)] : 0;
  const p50 = pick(0.5);
  const p95 = pick(0.95);
  const p99 = pick(0.99);
  const min = lats[0] ?? 0;
  const max = lats[lats.length - 1] ?? 0;
  const avg =
    lats.length > 0
      ? Math.round(lats.reduce((x, y) => x + y, 0) / lats.length)
      : 0;
  return (
    <div className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-4">
      <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)] mb-2.5">
        <Gauge className="size-3" />
        Latency · fleet
      </div>
      <div className="grid grid-cols-3 gap-2">
        <FleetLatChip label="P50" value={p50} tone="ok" />
        <FleetLatChip label="P95" value={p95} tone="warn" />
        <FleetLatChip label="P99" value={p99} tone="bad" />
      </div>
      <div className="mt-3 pt-3 border-t border-white/5 grid grid-cols-3 gap-2 text-[11px]">
        <FleetStat label="Avg" value={avg ? `${avg}ms` : "—"} />
        <FleetStat label="Min" value={min ? `${min}ms` : "—"} />
        <FleetStat label="Max" value={max ? `${max}ms` : "—"} />
      </div>
      <p className="mt-2 text-[10px] text-[var(--orbit-muted)]">
        Aggregated over every sampled P95 across visible projects.
      </p>
    </div>
  );
}

function FleetLatChip({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: "ok" | "warn" | "bad";
}) {
  const cls = {
    ok: "text-emerald-300",
    warn: "text-amber-300",
    bad: "text-rose-300",
  }[tone];
  return (
    <div className="rounded-lg border border-white/8 bg-white/3 px-2.5 py-2">
      <div className="text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)]">
        {label}
      </div>
      <div className={cn("mt-0.5 text-[15px] font-mono font-semibold", cls)}>
        {value > 0 ? value : "—"}
        {value > 0 && (
          <span className="text-[10px] text-[var(--orbit-muted)] ml-0.5">ms</span>
        )}
      </div>
    </div>
  );
}

function FleetStat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[9px] uppercase tracking-widest text-[var(--orbit-subtle)]">
        {label}
      </div>
      <div className="font-mono text-[var(--orbit-text)] text-[12px] mt-0.5">
        {value}
      </div>
    </div>
  );
}

function AggregateVolumeCard({
  visible,
  samples,
}: {
  visible: Project[];
  samples: Record<string, MetricSample[]>;
}) {
  const map = new Map<number, number>();
  visible.forEach((p) => {
    (samples[p.id] ?? []).forEach((s) => {
      map.set(s.ts, (map.get(s.ts) ?? 0) + s.reqCount);
    });
  });
  const data = Array.from(map.entries())
    .sort((a, b) => a[0] - b[0])
    .map(([ts, reqCount]) => ({ ts, reqCount }));
  const total = data.reduce((a, d) => a + d.reqCount, 0);
  return (
    <div className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-4">
      <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)] mb-2">
        <Zap className="size-3" />
        Volume · fleet
      </div>
      <div className="flex items-baseline gap-2">
        <span className="text-2xl font-mono font-semibold">{total}</span>
        <span className="text-[11px] text-[var(--orbit-muted)]">requests</span>
      </div>
      <div className="mt-2 h-[80px]">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data}>
            <defs>
              <linearGradient id="vol-fleet" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#a78bfa" stopOpacity={0.45} />
                <stop offset="100%" stopColor="#a78bfa" stopOpacity={0} />
              </linearGradient>
            </defs>
            <Area
              type="monotone"
              dataKey="reqCount"
              stroke="#a78bfa"
              fill="url(#vol-fleet)"
              strokeWidth={1.4}
              isAnimationActive={false}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

// ─── Top rankings ────────────────────────────────────────────────────────

type RankKey = "rps" | "p95" | "errs";

function rankProjects(
  projects: Project[],
  samples: Record<string, MetricSample[]>,
  key: RankKey,
  n: number,
) {
  return projects
    .map((p) => {
      const buf = samples[p.id] ?? [];
      const last = buf[buf.length - 1];
      const value = !last
        ? 0
        : key === "rps"
          ? last.reqCount / 5
          : key === "p95"
            ? last.p95Ms
            : last.errCount;
      return { project: p, value };
    })
    .filter((x) => x.value > 0)
    .sort((a, b) => b.value - a.value)
    .slice(0, n);
}

function TopProjectsCard({
  title,
  subtitle,
  icon,
  color,
  ranking,
  samples,
  valueKey,
  onSelect,
}: {
  title: string;
  subtitle: string;
  icon: React.ReactNode;
  color: string;
  ranking: { project: Project; value: number }[];
  samples: Record<string, MetricSample[]>;
  valueKey: RankKey;
  onSelect: (id: string) => void;
}) {
  const formatVal = (v: number) =>
    valueKey === "rps"
      ? `${v.toFixed(1)} req/s`
      : valueKey === "p95"
        ? `${v} ms`
        : String(v);
  return (
    <section className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-5 flex flex-col">
      <header className="flex items-center justify-between mb-3">
        <div>
          <h2 className="text-sm font-semibold flex items-center gap-1.5">
            {icon}
            {title}
          </h2>
          <p className="text-[11px] text-[var(--orbit-muted)] mt-0.5">{subtitle}</p>
        </div>
      </header>
      {ranking.length === 0 ? (
        <p className="text-[12px] text-[var(--orbit-muted)] italic flex-1 flex items-center justify-center min-h-32">
          No data yet
        </p>
      ) : (
        <ul className="space-y-1">
          {ranking.map(({ project, value }) => {
            const buf = samples[project.id] ?? [];
            const sparkData = buf.map((s) => ({
              v:
                valueKey === "rps"
                  ? s.reqCount / 5
                  : valueKey === "p95"
                    ? s.p95Ms
                    : s.errCount,
            }));
            return (
              <li key={project.id}>
                <button
                  onClick={() => onSelect(project.id)}
                  className="w-full grid grid-cols-[1fr_60px_auto] items-center gap-2 px-2 py-1.5 rounded-md hover:bg-white/5 transition-colors text-left"
                >
                  <span className="text-[12px] font-mono truncate flex items-center gap-1">
                    <span
                      className="size-1.5 rounded-full shrink-0"
                      style={{ background: color }}
                    />
                    {project.name}
                    <ArrowUpRight className="size-3 text-[var(--orbit-muted)] opacity-0 group-hover:opacity-100" />
                  </span>
                  <div className="h-5">
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart data={sparkData}>
                        <Line
                          type="monotone"
                          dataKey="v"
                          stroke={color}
                          strokeWidth={1.2}
                          dot={false}
                          isAnimationActive={false}
                        />
                      </LineChart>
                    </ResponsiveContainer>
                  </div>
                  <span
                    className="text-[11px] font-mono tabular-nums"
                    style={{ color }}
                  >
                    {formatVal(value)}
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

// ─── Fleet Health Table ──────────────────────────────────────────────────

function FleetHealthTable({
  projects,
  samples,
  onSelect,
}: {
  projects: Project[];
  samples: Record<string, MetricSample[]>;
  onSelect: (id: string) => void;
}) {
  return (
    <section className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-5 space-y-3">
      <header>
        <h2 className="text-sm font-semibold">Fleet health</h2>
        <p className="text-[11px] text-[var(--orbit-muted)] mt-0.5">
          Latest sample per project. Click a row to drill in.
        </p>
      </header>
      <div className="overflow-auto scrollbar-thin">
        <table className="w-full text-[12px]">
          <thead>
            <tr className="text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)]">
              <th className="text-left font-semibold py-2 px-2">Project</th>
              <th className="text-left font-semibold py-2 px-2">Status</th>
              <th className="text-right font-semibold py-2 px-2">Conns</th>
              <th className="text-right font-semibold py-2 px-2">RPS</th>
              <th className="text-right font-semibold py-2 px-2">P95</th>
              <th className="text-right font-semibold py-2 px-2">Errors</th>
              <th className="text-right font-semibold py-2 px-2">Mem</th>
              <th className="text-right font-semibold py-2 px-2">CPU</th>
            </tr>
          </thead>
          <tbody>
            {projects.map((p) => {
              const buf = samples[p.id] ?? [];
              const last = buf[buf.length - 1];
              return (
                <tr
                  key={p.id}
                  onClick={() => onSelect(p.id)}
                  className="cursor-pointer border-t border-white/5 hover:bg-white/4 transition-colors"
                >
                  <td className="py-2 px-2 font-mono">{p.name}</td>
                  <td className="py-2 px-2 capitalize">
                    {last?.status ?? "stopped"}
                  </td>
                  <td className="py-2 px-2 text-right font-mono">
                    {last?.conns ?? 0}
                  </td>
                  <td className="py-2 px-2 text-right font-mono">
                    {last ? (last.reqCount / 5).toFixed(1) : "0"}
                  </td>
                  <td className="py-2 px-2 text-right font-mono">
                    {last && last.p95Ms > 0 ? `${last.p95Ms}ms` : "—"}
                  </td>
                  <td
                    className={cn(
                      "py-2 px-2 text-right font-mono",
                      last && last.errCount > 0 && "text-rose-300",
                    )}
                  >
                    {last?.errCount ?? 0}
                  </td>
                  <td className="py-2 px-2 text-right font-mono">
                    {last && last.memKb > 0 ? formatKB(last.memKb) : "—"}
                  </td>
                  <td className="py-2 px-2 text-right font-mono">
                    {last && last.cpuPct > 0
                      ? `${last.cpuPct.toFixed(1)}%`
                      : "—"}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function formatKB(kb: number): string {
  if (kb < 1024) return `${kb} KB`;
  if (kb < 1024 * 1024) return `${(kb / 1024).toFixed(1)} MB`;
  return `${(kb / (1024 * 1024)).toFixed(2)} GB`;
}

// ─── Aggregate Heatmap ───────────────────────────────────────────────────

function UsageHeatmapAggregate({
  samples,
}: {
  samples: Record<string, MetricSample[]>;
}) {
  const grid = useMemo(() => {
    const cells: number[][] = Array.from({ length: 7 }, () =>
      Array(24).fill(0),
    );
    let max = 0;
    Object.values(samples).forEach((arr) => {
      arr.forEach((s) => {
        const d = new Date(s.ts * 1000);
        cells[d.getDay()][d.getHours()] += s.reqCount;
      });
    });
    for (const row of cells) for (const v of row) if (v > max) max = v;
    return { cells, max };
  }, [samples]);
  const days = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  const peak = (() => {
    let best = { day: 0, hour: 0, v: 0 };
    grid.cells.forEach((row, di) =>
      row.forEach((v, hi) => {
        if (v > best.v) best = { day: di, hour: hi, v };
      }),
    );
    return best;
  })();
  return (
    <section className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-5 space-y-3">
      <header className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold">Usage by hour</h2>
          <p className="text-[11px] text-[var(--orbit-muted)] mt-0.5">
            Combined request density across all projects · DOW × hour
          </p>
        </div>
        {grid.max > 0 && (
          <div className="text-[11px] text-[var(--orbit-muted)]">
            Peak{" "}
            <span className="text-[var(--orbit-text)] font-mono">
              {days[peak.day]} {String(peak.hour).padStart(2, "0")}:00
            </span>{" "}
            · {peak.v} reqs
          </div>
        )}
      </header>
      <div className="grid grid-cols-[2.5rem_1fr] gap-2">
        <div />
        <div
          className="grid text-[9px] text-[var(--orbit-subtle)]"
          style={{ gridTemplateColumns: "repeat(24, minmax(0, 1fr))" }}
        >
          {Array.from({ length: 24 }).map((_, h) => (
            <div key={h} className="text-center">
              {h % 6 === 0 ? String(h).padStart(2, "0") : ""}
            </div>
          ))}
        </div>
        {days.map((d, di) => (
          <Cells key={d} day={d} row={grid.cells[di]} max={grid.max} />
        ))}
      </div>
    </section>
  );
}

function Cells({
  day,
  row,
  max,
}: {
  day: string;
  row: number[];
  max: number;
}) {
  return (
    <>
      <div className="text-[10px] text-[var(--orbit-muted)] flex items-center">
        {day}
      </div>
      <div
        className="grid gap-[2px]"
        style={{ gridTemplateColumns: "repeat(24, minmax(0, 1fr))" }}
      >
        {row.map((v, h) => {
          const intensity = max > 0 ? v / max : 0;
          const bg =
            v === 0
              ? "rgba(255,255,255,0.04)"
              : `rgba(167, 139, 250, ${0.15 + intensity * 0.75})`;
          return (
            <div
              key={h}
              title={`${day} ${String(h).padStart(2, "0")}:00 · ${v} reqs`}
              className="aspect-square rounded-[2px]"
              style={{ background: bg }}
            />
          );
        })}
      </div>
    </>
  );
}

// FleetSummaryStrip renders a real time-series chart with the headline fleet
// metrics (running count, connections, RPS, restarts) stacked over the same
// time axis. Current values appear as a small legend up top.
function FleetSummaryStrip({
  projects,
  totals,
  samples,
  visibleProjects,
}: {
  projects: Project[];
  totals: { running: number; conns: number; restarts: number; rps: number };
  samples: Record<string, MetricSample[]>;
  visibleProjects: Project[];
}) {
  const series = useMemo(() => {
    const tsSet = new Set<number>();
    visibleProjects.forEach((p) =>
      (samples[p.id] ?? []).forEach((s) => tsSet.add(s.ts)),
    );
    const ts = Array.from(tsSet).sort((a, b) => a - b);
    return ts.map((t) => {
      let running = 0;
      let conns = 0;
      let reqs = 0;
      let restarts = 0;
      visibleProjects.forEach((p) => {
        const s = (samples[p.id] ?? []).find((x) => x.ts === t);
        if (!s) return;
        if (s.status === "running" || s.status === "starting") running++;
        conns += s.conns;
        reqs += s.reqCount;
        restarts += s.attempts;
      });
      return { ts: t, running, conns, rps: reqs / 5, restarts };
    });
  }, [visibleProjects, samples]);

  const headline: {
    label: string;
    icon: React.ReactNode;
    value: string;
    color: string;
  }[] = [
    {
      label: "Projects",
      icon: <Cpu className="size-3" />,
      value: String(projects.length),
      color: "#71717a",
    },
    {
      label: "Running",
      icon: <Activity className="size-3" />,
      value: String(totals.running),
      color: "#34d399",
    },
    {
      label: "Connections",
      icon: <Plug className="size-3" />,
      value: String(totals.conns),
      color: "#22d3ee",
    },
    {
      label: "RPS",
      icon: <Zap className="size-3" />,
      value: totals.rps > 0 ? totals.rps.toFixed(1) : "0",
      color: "#a78bfa",
    },
    {
      label: "Restarts",
      icon: <RotateCw className="size-3" />,
      value: String(totals.restarts),
      color: "#fbbf24",
    },
  ];

  return (
    <section className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-5 space-y-3">
      <header className="flex flex-wrap items-center gap-x-6 gap-y-2 justify-between">
        <div>
          <h2 className="text-sm font-semibold">Fleet activity</h2>
          <p className="text-[11px] text-[var(--orbit-muted)] mt-0.5">
            live
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-x-5 gap-y-1">
          {headline.map((h) => (
            <div key={h.label} className="flex items-center gap-2">
              <span
                className="size-1.5 rounded-full"
                style={{ background: h.color }}
              />
              <span className="text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)] inline-flex items-center gap-1">
                {h.icon}
                {h.label}
              </span>
              <span
                className="text-[13px] font-mono font-semibold tabular-nums"
                style={{ color: h.color }}
              >
                {h.value}
              </span>
            </div>
          ))}
        </div>
      </header>
      <ResponsiveContainer width="100%" height={180}>
        <LineChart
          data={series}
          margin={{ top: 5, right: 10, left: -10, bottom: 0 }}
        >
          <CartesianGrid stroke="rgba(255,255,255,0.05)" vertical={false} />
          <XAxis
            dataKey="ts"
            tickFormatter={tickTime}
            stroke="#52525b"
            fontSize={10}
          />
          <YAxis stroke="#52525b" fontSize={10} allowDecimals={false} />
          <Tooltip
            contentStyle={tooltipStyle}
            labelFormatter={(v) =>
              new Date(Number(v) * 1000).toLocaleTimeString()
            }
          />
          <Line
            type="monotone"
            dataKey="running"
            name="Running"
            stroke="#34d399"
            strokeWidth={1.5}
            dot={false}
            isAnimationActive={false}
          />
          <Line
            type="monotone"
            dataKey="conns"
            name="Connections"
            stroke="#22d3ee"
            strokeWidth={1.5}
            dot={false}
            isAnimationActive={false}
          />
          <Line
            type="monotone"
            dataKey="rps"
            name="RPS"
            stroke="#a78bfa"
            strokeWidth={1.5}
            dot={false}
            isAnimationActive={false}
          />
          <Line
            type="monotone"
            dataKey="restarts"
            name="Restarts"
            stroke="#fbbf24"
            strokeWidth={1.5}
            dot={false}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
    </section>
  );
}

// FleetStatusTimeline renders one colored strip per project showing the
// runtime state at every 5s sample. Lets the user eyeball the fleet's state
// over time without staring at numbers — green = running, cyan = starting,
// blue = idle, violet = suspended, rose = error, gray = stopped.
function FleetStatusTimeline({
  projects,
  samples,
  colorFor,
  onSelect,
}: {
  projects: Project[];
  samples: Record<string, MetricSample[]>;
  colorFor: Map<string, string>;
  onSelect: (id: string) => void;
}) {
  const statusBg: Record<string, string> = {
    running: "#34d399",
    starting: "#22d3ee",
    idle: "#60a5fa",
    suspended: "#a78bfa",
    stopped: "rgba(255,255,255,0.08)",
    error: "#fb7185",
  };
  const seen = new Set<string>();
  projects.forEach((p) =>
    (samples[p.id] ?? []).forEach((s) => seen.add(s.status)),
  );
  const labels: Record<string, string> = {
    running: "Running",
    starting: "Starting",
    idle: "Idle",
    suspended: "Suspended",
    stopped: "Stopped",
    error: "Error",
  };
  const fmt = (t: number) =>
    t > 0
      ? new Date(t * 1000).toLocaleTimeString([], {
          hour: "2-digit",
          minute: "2-digit",
        })
      : "—";

  // Window edges from any project that has samples.
  let first = 0;
  let last = 0;
  projects.forEach((p) => {
    const buf = samples[p.id] ?? [];
    if (buf.length > 0) {
      if (first === 0 || buf[0].ts < first) first = buf[0].ts;
      if (buf[buf.length - 1].ts > last) last = buf[buf.length - 1].ts;
    }
  });

  return (
    <section className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-5 space-y-3">
      <header className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold">Fleet status timeline</h2>
          <p className="text-[11px] text-[var(--orbit-muted)] mt-0.5">
            Runtime state per project at every 5s sample
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[10px]">
          {Object.entries(statusBg)
            .filter(([k]) => seen.has(k))
            .map(([k, v]) => (
              <span key={k} className="inline-flex items-center gap-1.5">
                <span className="size-2 rounded-sm" style={{ background: v }} />
                <span className="text-[var(--orbit-muted)]">{labels[k]}</span>
              </span>
            ))}
        </div>
      </header>

      {projects.length === 0 ? (
        <p className="text-[11px] text-[var(--orbit-muted)] italic">
          No projects.
        </p>
      ) : (
        <>
          <div className="space-y-1.5">
            {projects.map((p) => {
              const buf = samples[p.id] ?? [];
              return (
                <div
                  key={p.id}
                  className="grid grid-cols-[10rem_1fr] gap-3 items-center cursor-pointer hover:bg-white/3 rounded-md px-1 py-0.5 transition-colors"
                  onClick={() => onSelect(p.id)}
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <span
                      className="size-1.5 rounded-full shrink-0"
                      style={{ background: colorFor.get(p.id) }}
                    />
                    <span className="text-[12px] font-mono truncate">
                      {p.name}
                    </span>
                  </div>
                  {buf.length === 0 ? (
                    <div className="h-5 rounded-md bg-white/4" />
                  ) : (
                    <div className="flex gap-[1px] h-5 rounded-md overflow-hidden">
                      {buf.map((s, i) => (
                        <div
                          key={i}
                          className="flex-1 transition-opacity hover:opacity-80"
                          style={{
                            background:
                              statusBg[s.status] ?? statusBg.stopped,
                          }}
                          title={`${fmt(s.ts)} · ${labels[s.status] ?? s.status}`}
                        />
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
          <div className="grid grid-cols-[10rem_1fr] gap-3">
            <div />
            <div className="flex justify-between text-[10px] text-[var(--orbit-muted)] font-mono">
              <span>{fmt(first)}</span>
              <span>{fmt(last)}</span>
            </div>
          </div>
        </>
      )}
    </section>
  );
}

function KpiCard({
  icon,
  label,
  value,
  tone = "muted",
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  tone?: "ok" | "warn" | "active" | "muted";
}) {
  const cls = {
    ok: "text-emerald-300",
    warn: "text-amber-300",
    active: "text-cyan-300",
    muted: "text-[var(--orbit-text)]",
  }[tone];
  return (
    <div className="rounded-xl border border-[var(--orbit-border)] bg-white/2 px-4 py-3.5">
      <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)]">
        {icon}
        {label}
      </div>
      <div className={cn("mt-1.5 text-2xl font-mono font-semibold", cls)}>
        {value}
      </div>
    </div>
  );
}

function ChartCard({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-5 space-y-3">
      <header>
        <h2 className="text-sm font-semibold">{title}</h2>
        {subtitle && (
          <p className="text-[11px] text-[var(--orbit-muted)] mt-0.5">{subtitle}</p>
        )}
      </header>
      {children}
    </section>
  );
}
