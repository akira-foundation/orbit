import { useEffect, useMemo, useState } from "react";
import { useInterval, useLocalStorage } from "usehooks-ts";
import { api } from "../api";
import { useProjects } from "../store";
import type { MetricSample } from "../types";
import { MetricsFilter, MetricInterval, getSinceTsForInterval } from "../components/MetricsFilter";
import { AttentionStrip } from "../components/MetricsAttention";
import { Skeleton } from "../components/ui/skeleton";
import {
  PALETTE,
  computeTotals,
  mergeForChart,
  mergeMemChart,
} from "../components/metrics/charts";
import { ProjectFilter } from "../components/metrics/ProjectFilter";
import {
  AggregateStatusCard,
  AggregateLatencyCard,
  AggregateVolumeCard,
} from "../components/metrics/AggregateCards";
import { FleetSummaryStrip } from "../components/metrics/FleetSummaryStrip";
import { TimeSeriesGrid } from "../components/metrics/TimeSeriesGrid";
import { FleetHealthTable } from "../components/metrics/FleetHealthTable";

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

        <AttentionStrip
          projects={projects}
          samples={samples}
          onSelect={showProjectMetrics}
        />

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

        <TimeSeriesGrid
          merged={merged}
          memMerged={memMerged}
          visibleProjects={visibleProjects}
          colorFor={colorFor}
          samples={samples}
        />

        <FleetHealthTable
          projects={visibleProjects}
          samples={samples}
          onSelect={showProjectMetrics}
        />
      </div>
    </div>
  );
}


