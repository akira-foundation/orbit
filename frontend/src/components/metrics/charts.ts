import type { CSSProperties } from "react";
import type { MetricSample, Project } from "../../types";

export const PALETTE = [
  "#22d3ee",
  "#a78bfa",
  "#34d399",
  "#f472b6",
  "#fbbf24",
  "#60a5fa",
  "#fb7185",
  "#4ade80",
];

export const tooltipStyle: CSSProperties = {
  background: "rgba(20,21,28,0.95)",
  border: "1px solid rgba(255,255,255,0.08)",
  borderRadius: 8,
  fontSize: 11,
};

export function tickTime(v: number) {
  return new Date(v * 1000).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatKB(kb: number): string {
  if (kb < 1024) return `${kb} KB`;
  if (kb < 1024 * 1024) return `${(kb / 1024).toFixed(1)} MB`;
  return `${(kb / (1024 * 1024)).toFixed(2)} GB`;
}

export function computeTotals(
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

export function mergeMemChart(
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

export function mergeForChart(
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

export function aggregate(
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
