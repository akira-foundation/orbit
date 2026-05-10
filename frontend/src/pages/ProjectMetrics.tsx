import { useEffect, useMemo, useState } from "react";
import { useInterval } from "usehooks-ts";
import {
  Area,
  AreaChart,
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
  ArrowLeft,
  Cpu,
  Gauge,
  HardDrive,
  HelpCircle,
  MemoryStick,
  Plug,
  Timer,
  Zap,
} from "lucide-react";
import { api } from "../api";
import { useProjects } from "../store";
import type { MetricSample, Project } from "../types";
import { cn } from "../lib/cn";

const ACCENT = "#22d3ee";

export function ProjectMetricsPage({ id }: { id: string }) {
  const { projects, select } = useProjects();
  const project = useMemo(
    () => projects.find((p) => p.id === id) ?? null,
    [projects, id],
  );
  const [samples, setSamples] = useState<MetricSample[]>([]);

  const refresh = async () => {
    try {
      setSamples(await api.runtimeMetrics(id));
    } catch {
      /* ignore */
    }
  };

  useEffect(() => {
    refresh();
  }, [id]);
  useInterval(refresh, 5000);

  if (!project) {
    return (
      <div className="h-full flex items-center justify-center">
        <p className="text-sm text-[var(--orbit-muted)]">Project not found.</p>
      </div>
    );
  }

  const last = samples[samples.length - 1];
  const SAMPLE_S = 5; // matches backend sampleInterval
  const rps = last ? last.reqCount / SAMPLE_S : 0;
  const errPct =
    last && last.reqCount > 0 ? (last.errCount / last.reqCount) * 100 : 0;

  return (
    <div className="h-full overflow-auto scrollbar-thin">
      <div className="mx-auto w-full max-w-6xl px-10 py-8 space-y-8">
        <header className="flex items-center justify-between">
          <div className="space-y-1">
            <button
              onClick={() => select(id)}
              className="inline-flex items-center gap-1.5 text-[11px] text-[var(--orbit-muted)] hover:text-[var(--orbit-text)]"
            >
              <ArrowLeft className="size-3" />
              Back to {project.name}
            </button>
            <h1 className="text-2xl font-semibold tracking-tight">
              {project.name} · Metrics
            </h1>
            <p className="text-xs text-[var(--orbit-muted)]">
              Last 30 minutes · 5s sampling · live
            </p>
          </div>
        </header>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <ResponseStatusCard samples={samples} />
          <LatencyCard samples={samples} />
          <VolumeCard samples={samples} />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <ResourceCard samples={samples} />

        <ChartCard
          title="Uptime vs downtime"
          subtitle="Cumulative seconds alive vs seconds stopped / errored / idle"
        >
          <ResponsiveContainer width="100%" height={220}>
            <LineChart
              data={(() => {
                // Walk samples in order, accumulating seconds in each
                // category. Sample interval is 5s, so each sample adds 5s
                // to one bucket depending on its status.
                const SAMPLE_S = 5;
                let upS = 0;
                let downS = 0;
                return samples.map((s) => {
                  const isUp = s.status === "running" || s.status === "starting";
                  if (isUp) upS += SAMPLE_S;
                  else downS += SAMPLE_S;
                  return { ts: s.ts, upS, downS };
                });
              })()}
              margin={{ top: 5, right: 10, left: -10, bottom: 0 }}
            >
              <CartesianGrid stroke="rgba(255,255,255,0.05)" vertical={false} />
              <XAxis dataKey="ts" tickFormatter={tickTime} stroke="#52525b" fontSize={10} />
              <YAxis
                stroke="#52525b"
                fontSize={10}
                tickFormatter={(v) => `${Math.round(v)}s`}
              />
              <Tooltip
                contentStyle={tooltipStyle}
                labelFormatter={(v) => new Date(Number(v) * 1000).toLocaleTimeString()}
                formatter={(v, name) => [`${Math.round(Number(v))}s`, name]}
              />
              <Line
                type="monotone"
                dataKey="upS"
                name="Uptime"
                stroke={ACCENT}
                strokeWidth={1.5}
                dot={false}
                isAnimationActive={false}
                connectNulls
              />
              <Line
                type="monotone"
                dataKey="downS"
                name="Downtime"
                stroke="#fb7185"
                strokeWidth={1.5}
                dot={false}
                isAnimationActive={false}
                connectNulls
              />
            </LineChart>
          </ResponsiveContainer>
          <div className="flex flex-wrap gap-x-4 gap-y-1 text-[10px] text-[var(--orbit-muted)] pt-2 border-t border-white/5">
            <Legend dot={ACCENT} label="Uptime" hint="running + starting" />
            <Legend dot="#fb7185" label="Downtime" hint="stopped / error / idle" />
          </div>
        </ChartCard>
        </div>

        <StatusTimeline samples={samples} />

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <ChartCard
          title="HTTP vs WebSocket"
          subtitle="Request mix per sample window"
        >
          <ResponsiveContainer width="100%" height={180}>
            <LineChart data={samples} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
              <CartesianGrid stroke="rgba(255,255,255,0.05)" vertical={false} />
              <XAxis dataKey="ts" tickFormatter={tickTime} stroke="#52525b" fontSize={10} />
              <YAxis stroke="#52525b" fontSize={10} allowDecimals={false} />
              <Tooltip contentStyle={tooltipStyle} labelFormatter={(v) => new Date(Number(v) * 1000).toLocaleTimeString()} />
              <Line type="monotone" dataKey="httpReqs" name="HTTP" stroke="#22d3ee" strokeWidth={1.4} dot={false} isAnimationActive={false} connectNulls />
              <Line type="monotone" dataKey="wsReqs" name="WebSocket" stroke="#a78bfa" strokeWidth={1.4} dot={false} isAnimationActive={false} connectNulls />
            </LineChart>
          </ResponsiveContainer>
        </ChartCard>

        <ChartCard
          title="Active connections"
          subtitle="Open HTTP / WebSocket proxy connections"
        >
          <ResponsiveContainer width="100%" height={200}>
            <AreaChart
              data={samples}
              margin={{ top: 5, right: 10, left: -10, bottom: 0 }}
            >
              <defs>
                <linearGradient id="conns-grad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={ACCENT} stopOpacity={0.5} />
                  <stop offset="100%" stopColor={ACCENT} stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid stroke="rgba(255,255,255,0.05)" vertical={false} />
              <XAxis dataKey="ts" tickFormatter={tickTime} stroke="#52525b" fontSize={10} />
              <YAxis stroke="#52525b" fontSize={10} allowDecimals={false} />
              <Tooltip
                contentStyle={tooltipStyle}
                labelFormatter={(v) => new Date(Number(v) * 1000).toLocaleTimeString()}
              />
              <Area
                type="monotone"
                dataKey="conns"
                stroke={ACCENT}
                fill="url(#conns-grad)"
                strokeWidth={1.5}
                isAnimationActive={false}
                connectNulls
              />
            </AreaChart>
          </ResponsiveContainer>
        </ChartCard>

        </div>
      </div>
    </div>
  );
}

// ResourceCard splits memory and CPU into two small filled-area sub-charts
// stacked inside one card. Decoupled scales so a memory spike doesn't crush
// the CPU line into the floor (or vice versa).
function ResourceCard({ samples }: { samples: MetricSample[] }) {
  const last = samples[samples.length - 1];
  return (
    <section className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-5 space-y-3">
      <header className="flex items-baseline justify-between gap-3">
        <div>
          <h2 className="text-sm font-semibold">Process resources</h2>
          <p className="text-[11px] text-[var(--orbit-muted)] mt-0.5">
            RSS memory + CPU% across the dev process group
          </p>
        </div>
        <div className="flex items-center gap-4 text-[11px]">
          <span className="text-cyan-300 font-mono">
            {last && last.memKb > 0 ? formatKB(last.memKb) : "—"}
          </span>
          <span className="text-amber-300 font-mono">
            {last && last.cpuPct > 0 ? `${last.cpuPct.toFixed(1)}%` : "—"}
          </span>
        </div>
      </header>

      <div>
        <div className="flex items-center justify-between text-[10px] text-[var(--orbit-muted)] mb-1">
          <span className="inline-flex items-center gap-1.5">
            <span className="size-1.5 rounded-full bg-cyan-300" />
            RSS memory
          </span>
        </div>
        <ResponsiveContainer width="100%" height={110}>
          <AreaChart data={samples} margin={{ top: 0, right: 0, left: -10, bottom: 0 }}>
            <defs>
              <linearGradient id="mem-grad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#22d3ee" stopOpacity={0.45} />
                <stop offset="100%" stopColor="#22d3ee" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid stroke="rgba(255,255,255,0.05)" vertical={false} />
            <XAxis dataKey="ts" tickFormatter={tickTime} stroke="#52525b" fontSize={9} />
            <YAxis stroke="#52525b" fontSize={9} tickFormatter={(v) => formatKB(Number(v))} width={60} />
            <Tooltip
              contentStyle={tooltipStyle}
              labelFormatter={(v) => new Date(Number(v) * 1000).toLocaleTimeString()}
              formatter={(v) => formatKB(Number(v))}
            />
            <Area
              type="monotone"
              dataKey="memKb"
              stroke="#22d3ee"
              fill="url(#mem-grad)"
              strokeWidth={1.4}
              isAnimationActive={false}
              connectNulls
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      <div>
        <div className="flex items-center justify-between text-[10px] text-[var(--orbit-muted)] mb-1">
          <span className="inline-flex items-center gap-1.5">
            <span className="size-1.5 rounded-full bg-amber-300" />
            CPU%
          </span>
        </div>
        <ResponsiveContainer width="100%" height={90}>
          <AreaChart data={samples} margin={{ top: 0, right: 0, left: -10, bottom: 0 }}>
            <defs>
              <linearGradient id="cpu-grad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#fbbf24" stopOpacity={0.45} />
                <stop offset="100%" stopColor="#fbbf24" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid stroke="rgba(255,255,255,0.05)" vertical={false} />
            <XAxis dataKey="ts" tickFormatter={tickTime} stroke="#52525b" fontSize={9} />
            <YAxis stroke="#52525b" fontSize={9} tickFormatter={(v) => `${v}%`} width={60} />
            <Tooltip
              contentStyle={tooltipStyle}
              labelFormatter={(v) => new Date(Number(v) * 1000).toLocaleTimeString()}
              formatter={(v) => `${Number(v).toFixed(1)}%`}
            />
            <Area
              type="monotone"
              dataKey="cpuPct"
              stroke="#fbbf24"
              fill="url(#cpu-grad)"
              strokeWidth={1.4}
              isAnimationActive={false}
              connectNulls
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </section>
  );
}

// StatusTimeline draws one horizontal strip showing the runtime's state over
// time. Each sample becomes a colored block (running / starting / error /
// stopped / idle). Lets the user see at a glance "I had 3 minutes of running,
// then it crashed, restarted, idle, then auto-stopped". More valuable for
// dev workflow than a mixed-scale traffic chart.
function StatusTimeline({ samples }: { samples: MetricSample[] }) {
  const colors: Record<string, { bg: string; label: string; text: string }> = {
    running:   { bg: "#34d399", text: "text-emerald-300", label: "Running" },
    starting:  { bg: "#22d3ee", text: "text-cyan-300",    label: "Starting" },
    idle:      { bg: "#60a5fa", text: "text-sky-300",     label: "Idle" },
    suspended: { bg: "#a78bfa", text: "text-violet-300",  label: "Suspended" },
    stopped:   { bg: "rgba(255,255,255,0.08)", text: "text-[var(--orbit-muted)]", label: "Stopped" },
    error:     { bg: "#fb7185", text: "text-rose-300",    label: "Error" },
  };
  const seen = new Set<string>(samples.map((s) => s.status as string));

  // Bucket counts for "time spent in state" summary.
  const counts: Record<string, number> = {};
  samples.forEach((s) => {
    counts[s.status] = (counts[s.status] ?? 0) + 1;
  });

  // Sample interval is 5s — convert counts into seconds for the summary.
  const SAMPLE_S = 5;

  const lastStatus = samples[samples.length - 1]?.status;
  const lastColor = lastStatus ? colors[lastStatus] : null;

  const first = samples[0]?.ts ?? 0;
  const last = samples[samples.length - 1]?.ts ?? 0;
  const fmt = (t: number) =>
    t > 0
      ? new Date(t * 1000).toLocaleTimeString([], {
          hour: "2-digit",
          minute: "2-digit",
        })
      : "—";

  const formatSec = (s: number) => {
    if (s < 60) return `${s}s`;
    const m = Math.floor(s / 60);
    const sec = s % 60;
    if (m < 60) return sec ? `${m}m ${sec}s` : `${m}m`;
    const h = Math.floor(m / 60);
    return `${h}h ${m % 60}m`;
  };

  return (
    <section className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-5 space-y-3">
      <header className="flex items-baseline justify-between gap-3">
        <div>
          <h2 className="text-sm font-semibold">Status timeline</h2>
          <p className="text-[11px] text-[var(--orbit-muted)] mt-0.5">
            Runtime state at every 5s sample over the last 30 minutes
          </p>
        </div>
        {lastColor && (
          <span
            className={cn(
              "text-[11px] font-mono inline-flex items-center gap-1.5",
              lastColor.text,
            )}
          >
            <span
              className="size-1.5 rounded-full"
              style={{ background: lastColor.bg }}
            />
            {lastColor.label}
          </span>
        )}
      </header>

      {samples.length === 0 ? (
        <p className="text-[11px] text-[var(--orbit-muted)] italic">
          No samples yet.
        </p>
      ) : (
        <>
          <div>
            <div className="flex items-center justify-between text-[10px] text-[var(--orbit-muted)] mb-1">
              <span className="inline-flex items-center gap-1.5">
                <span className="size-1.5 rounded-full bg-emerald-300" />
                Runtime state
              </span>
            </div>
            <div className="flex gap-[1px] h-8 rounded-md overflow-hidden">
              {samples.map((s, i) => {
                const c = colors[s.status] ?? colors.stopped;
                return (
                  <div
                    key={i}
                    className="flex-1 transition-opacity hover:opacity-80"
                    style={{ background: c.bg }}
                    title={`${fmt(s.ts)} · ${c.label}${s.port ? ` · :${s.port}` : ""}${s.attempts ? ` · attempts: ${s.attempts}` : ""}`}
                  />
                );
              })}
            </div>
            <div className="flex justify-between text-[10px] text-[var(--orbit-muted)] font-mono mt-1">
              <span>{fmt(first)}</span>
              <span>{fmt(last)}</span>
            </div>
          </div>

          <div className="pt-2 border-t border-white/5 flex flex-wrap gap-x-4 gap-y-1 text-[10px]">
            {Object.entries(colors)
              .filter(([k]) => seen.has(k))
              .map(([k, v]) => (
                <span key={k} className="inline-flex items-center gap-1.5">
                  <span
                    className="size-1.5 rounded-full"
                    style={{ background: v.bg }}
                  />
                  <span className="text-[var(--orbit-text)]">{v.label}</span>
                  <span className="text-[var(--orbit-muted)]">
                    · {formatSec((counts[k] ?? 0) * SAMPLE_S)}
                  </span>
                </span>
              ))}
          </div>
        </>
      )}
    </section>
  );
}

function Legend({
  dot,
  label,
  hint,
  dashed,
}: {
  dot: string;
  label: string;
  hint?: string;
  dashed?: boolean;
}) {
  return (
    <span className="inline-flex items-center gap-1.5">
      {dashed ? (
        <span
          className="inline-block w-3 h-[2px] rounded-full"
          style={{
            background: `repeating-linear-gradient(90deg, ${dot} 0 3px, transparent 3px 5px)`,
          }}
        />
      ) : (
        <span className="size-1.5 rounded-full" style={{ background: dot }} />
      )}
      <span className="text-[var(--orbit-text)]">{label}</span>
      {hint && <span>· {hint}</span>}
    </span>
  );
}

function ResponseStatusCard({ samples }: { samples: MetricSample[] }) {
  const totalReq = samples.reduce((a, s) => a + s.reqCount, 0);
  const totalErr = samples.reduce((a, s) => a + s.errCount, 0);
  const totalOk = Math.max(totalReq - totalErr, 0);
  const errRate = totalReq > 0 ? (totalErr / totalReq) * 100 : 0;
  const data = totalReq > 0 ? [
    { name: "2XX/3XX", value: totalOk, color: "#34d399" },
    { name: "4XX/5XX", value: totalErr, color: "#fbbf24" },
  ] : [{ name: "—", value: 1, color: "rgba(255,255,255,0.06)" }];

  return (
    <div className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-4">
      <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)] mb-2">
        <Activity className="size-3" />
        Response status
      </div>
      <div className="flex items-center gap-4">
        <div className="relative size-[110px] shrink-0">
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={data}
                innerRadius={36}
                outerRadius={52}
                paddingAngle={totalReq > 0 ? 2 : 0}
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
            <span className="text-xl font-mono font-semibold">{totalReq}</span>
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
            <span className="font-mono text-[var(--orbit-text)]">{totalOk}</span>
          </div>
          <div className="flex items-center justify-between gap-3">
            <span className="inline-flex items-center gap-1.5">
              <span className="size-1.5 rounded-full bg-amber-400" />
              4xx / 5xx
            </span>
            <span className="font-mono text-[var(--orbit-text)]">{totalErr}</span>
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

function LatencyCard({ samples }: { samples: MetricSample[] }) {
  const last = samples[samples.length - 1];
  const allLat = samples.flatMap((s) =>
    [s.p50Ms, s.p95Ms, s.p99Ms].filter((x) => x > 0),
  );
  const min = allLat.length ? Math.min(...allLat) : 0;
  const max = allLat.length ? Math.max(...allLat) : 0;
  const avg =
    allLat.length > 0
      ? Math.round(allLat.reduce((a, b) => a + b, 0) / allLat.length)
      : 0;

  return (
    <div className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-4">
      <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)] mb-2.5">
        <Gauge className="size-3" />
        Latency
      </div>
      <div className="grid grid-cols-3 gap-2">
        <LatencyChip label="P50" value={last?.p50Ms ?? 0} tone="ok" />
        <LatencyChip label="P95" value={last?.p95Ms ?? 0} tone="warn" />
        <LatencyChip label="P99" value={last?.p99Ms ?? 0} tone="bad" />
      </div>
      <div className="mt-3 pt-3 border-t border-white/5 grid grid-cols-3 gap-2 text-[11px]">
        <Stat3 label="Avg" value={avg ? `${avg}ms` : "—"} />
        <Stat3 label="Min" value={min ? `${min}ms` : "—"} />
        <Stat3 label="Max" value={max ? `${max}ms` : "—"} />
      </div>
      <p className="mt-2 text-[10px] text-[var(--orbit-muted)]">
        <span className="text-emerald-400">P50</span> typical ·{" "}
        <span className="text-amber-300">P95</span> slow tail ·{" "}
        <span className="text-rose-300">P99</span> worst case
      </p>
    </div>
  );
}

function LatencyChip({
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
          <span className="text-[10px] text-[var(--orbit-muted)] ml-0.5">
            ms
          </span>
        )}
      </div>
    </div>
  );
}

function Stat3({ label, value }: { label: string; value: string }) {
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

function VolumeCard({ samples }: { samples: MetricSample[] }) {
  const totalReq = samples.reduce((a, s) => a + s.reqCount, 0);
  return (
    <div className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-4">
      <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)] mb-2">
        <Zap className="size-3" />
        Volume
      </div>
      <div className="flex items-baseline gap-2">
        <span className="text-2xl font-mono font-semibold">{totalReq}</span>
        <span className="text-[11px] text-[var(--orbit-muted)]">requests</span>
      </div>
      <div className="mt-2 h-[80px]">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={samples}>
            <defs>
              <linearGradient id="vol-grad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#a78bfa" stopOpacity={0.45} />
                <stop offset="100%" stopColor="#a78bfa" stopOpacity={0} />
              </linearGradient>
            </defs>
            <Area
              type="monotone"
              dataKey="reqCount"
              stroke="#a78bfa"
              fill="url(#vol-grad)"
              strokeWidth={1.4}
              isAnimationActive={false}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

// UsageHeatmap renders a 7×24 grid (day-of-week × hour-of-day) where each
// cell's opacity reflects request volume bucketed from samples. Lightweight
// — just divs, no chart lib. Mirrors the screenshot's "usage by hour" feel.
function UsageHeatmap({ samples }: { samples: MetricSample[] }) {
  const grid = useMemo(() => {
    const cells: number[][] = Array.from({ length: 7 }, () =>
      Array(24).fill(0),
    );
    let max = 0;
    for (const s of samples) {
      const d = new Date(s.ts * 1000);
      cells[d.getDay()][d.getHours()] += s.reqCount;
    }
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
            Request density across day-of-week × hour-of-day (within sampled
            window)
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
      <div className="flex items-center gap-2 text-[10px] text-[var(--orbit-muted)] pt-1">
        Less
        {[0.1, 0.3, 0.55, 0.8, 1].map((o) => (
          <span
            key={o}
            className="inline-block size-2 rounded-sm"
            style={{ background: `rgba(167, 139, 250, ${o})` }}
          />
        ))}
        More
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

function tickTime(v: number) {
  return new Date(v * 1000).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatKB(kb: number): string {
  if (kb < 1024) return `${kb} KB`;
  if (kb < 1024 * 1024) return `${(kb / 1024).toFixed(1)} MB`;
  return `${(kb / (1024 * 1024)).toFixed(2)} GB`;
}

function formatBytes(b: number): string {
  if (b < 1024) return `${b} B`;
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`;
  if (b < 1024 * 1024 * 1024) return `${(b / (1024 * 1024)).toFixed(1)} MB`;
  return `${(b / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function Glossary() {
  const items: { term: string; def: string }[] = [
    {
      term: "RPS",
      def: "Requests per second served by the proxy in the last 5s sample window.",
    },
    {
      term: "P50 (median)",
      def: "Half of requests finish faster than this latency; half slower.",
    },
    {
      term: "P95",
      def: "95th percentile latency. Only 5% of requests are slower than this — a good signal for tail latency.",
    },
    {
      term: "P99",
      def: "99th percentile. Worst-case latency for 99% of traffic; spikes here are felt by users on slow paths.",
    },
    {
      term: "Error %",
      def: "Share of HTTP responses with status ≥ 400 (4xx client errors + 5xx server errors).",
    },
    {
      term: "RSS (Memory)",
      def: "Resident Set Size — actual RAM the dev process group is holding. Summed across all child workers.",
    },
    {
      term: "CPU%",
      def: "CPU utilization across the process group. Percent of one core; can exceed 100% when workers spread across cores.",
    },
    {
      term: "Wake latency",
      def: "Time from the moment Orbit asked the runtime to start until it became healthy on a port.",
    },
    {
      term: "HTTP / WS",
      def: "HTTP requests vs WebSocket upgrades. WS connections stay open for the lifetime of the socket.",
    },
    {
      term: "Bandwidth",
      def: "Bytes transferred between the proxy and the dev server in each 5s window. Useful to spot heavy fetches or large bundles.",
    },
    {
      term: "Auto-stop",
      def: "Orbit stops a runtime that has had zero active connections for 10s, then re-wakes it on the next request.",
    },
    {
      term: "Crash",
      def: "A previously-running runtime exited unexpectedly. Self-heal kicks in with exponential backoff.",
    },
  ];
  return (
    <section className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-5">
      <header className="flex items-center gap-2 mb-3">
        <HelpCircle className="size-3.5 text-[var(--orbit-muted)]" />
        <h2 className="text-sm font-semibold">Glossary</h2>
      </header>
      <dl className="grid grid-cols-1 md:grid-cols-2 gap-x-6 gap-y-2.5 text-[12px]">
        {items.map((it) => (
          <div key={it.term} className="grid grid-cols-[7rem_1fr] gap-3">
            <dt className="font-mono text-[11px] text-[var(--orbit-text)]">
              {it.term}
            </dt>
            <dd className="text-[var(--orbit-muted)] leading-relaxed">
              {it.def}
            </dd>
          </div>
        ))}
      </dl>
    </section>
  );
}

function formatUptime(ms: number): string {
  const s = Math.floor(ms / 1000);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${sec}s`;
  return `${sec}s`;
}

const tooltipStyle: React.CSSProperties = {
  background: "rgba(20,21,28,0.95)",
  border: "1px solid rgba(255,255,255,0.08)",
  borderRadius: 8,
  fontSize: 11,
};

function Kpi({
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
      <div className={cn("mt-1.5 text-2xl font-mono font-semibold capitalize", cls)}>
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
          <p className="text-[11px] text-[var(--orbit-muted)] mt-0.5">
            {subtitle}
          </p>
        )}
      </header>
      {children}
    </section>
  );
}
