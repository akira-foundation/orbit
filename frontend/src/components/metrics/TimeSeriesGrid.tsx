import {
  Area,
  AreaChart,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { MetricSample, Project } from "../../types";
import { formatKB, tickTime, tooltipStyle } from "./charts";
import { ChartCard } from "./ChartCard";
import { UsageHeatmapAggregate } from "./UsageHeatmap";

export function TimeSeriesGrid({
  merged,
  memMerged,
  visibleProjects,
  colorFor,
  samples,
}: {
  merged: Array<Record<string, number>>;
  memMerged: Array<Record<string, number>>;
  visibleProjects: Project[];
  colorFor: Map<string, string>;
  samples: Record<string, MetricSample[]>;
}) {
  return (
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

      <ChartCard title="Memory" subtitle="RSS per project over time">
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
  );
}
