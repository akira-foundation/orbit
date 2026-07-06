import {
  Area,
  AreaChart,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
} from "recharts";
import { Activity, Gauge, Zap } from "lucide-react";
import type { MetricSample, Project } from "../../types";
import { cn } from "../../lib/cn";
import { aggregate } from "./charts";

export function AggregateStatusCard({
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

export function AggregateLatencyCard({
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

export function AggregateVolumeCard({
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
