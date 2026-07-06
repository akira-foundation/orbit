import { useMemo } from "react";
import type { MetricSample } from "../../types";

export function UsageHeatmapAggregate({
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
