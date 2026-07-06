import type { MetricSample, Project } from "../../types";
import { cn } from "../../lib/cn";
import { formatKB } from "./charts";

export function FleetHealthTable({
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
