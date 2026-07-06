import { useMemo } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Activity, Cpu, Plug, RotateCw, Zap } from "lucide-react";
import type { MetricSample, Project } from "../../types";
import { tickTime, tooltipStyle } from "./charts";

export function FleetSummaryStrip({
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
