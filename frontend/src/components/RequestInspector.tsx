import { useEffect, useState } from "react";
import { useInterval } from "usehooks-ts";
import { api } from "../api";
import type { RequestEntry } from "../types";
import { cn } from "../lib/cn";

function statusColor(s: number): string {
  if (s >= 500) return "text-rose-300";
  if (s >= 400) return "text-amber-300";
  if (s >= 300) return "text-sky-300";
  if (s >= 200) return "text-emerald-300";
  return "text-[var(--orbit-muted)]";
}

function methodColor(m: string): string {
  switch (m) {
    case "GET":
      return "text-emerald-300";
    case "POST":
      return "text-sky-300";
    case "PUT":
    case "PATCH":
      return "text-amber-300";
    case "DELETE":
      return "text-rose-300";
    default:
      return "text-[var(--orbit-muted)]";
  }
}

function formatSize(n: number): string {
  if (n <= 0) return "-";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

export function RequestInspector({ projectId }: { projectId: string }) {
  const [rows, setRows] = useState<RequestEntry[]>([]);
  const [live, setLive] = useState(true);

  const refresh = () => {
    api.projectRequests(projectId).then(setRows).catch(() => {});
  };

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  useInterval(refresh, live ? 1500 : null);

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-[11px] text-[var(--orbit-muted)]">
          Last {rows.length} requests proxied to this project (newest first).
        </p>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setLive((v) => !v)}
            className={cn(
              "h-7 px-2.5 rounded-md text-[11px] border transition-colors",
              live
                ? "border-emerald-400/30 bg-emerald-400/10 text-emerald-300"
                : "border-white/10 text-[var(--orbit-muted)] hover:text-[var(--orbit-text)]",
            )}
          >
            {live ? "Live" : "Paused"}
          </button>
          <button
            onClick={() => setRows([])}
            className="h-7 px-2.5 rounded-md text-[11px] border border-white/10 text-[var(--orbit-muted)] hover:text-[var(--orbit-text)]"
          >
            Clear
          </button>
        </div>
      </div>

      <div className="overflow-auto scrollbar-thin rounded-lg border border-[var(--orbit-border)] max-h-[60vh]">
        <table className="w-full text-[12px] font-mono">
          <thead className="sticky top-0 bg-[rgba(28,28,34,0.95)]">
            <tr className="text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)]">
              <th className="text-left font-semibold px-3 py-2">Method</th>
              <th className="text-right font-semibold px-3 py-2">Status</th>
              <th className="text-left font-semibold px-3 py-2">Path</th>
              <th className="text-right font-semibold px-3 py-2">Time</th>
              <th className="text-right font-semibold px-3 py-2">Size</th>
              <th className="text-right font-semibold px-3 py-2">At</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr key={i} className="border-t border-white/5 hover:bg-white/[0.03]">
                <td className={cn("px-3 py-1.5 font-semibold", methodColor(r.method))}>
                  {r.ws ? "WS" : r.method}
                </td>
                <td className={cn("px-3 py-1.5 text-right", statusColor(r.status))}>
                  {r.status}
                </td>
                <td className="px-3 py-1.5 truncate max-w-md text-[var(--orbit-text)]/85">
                  {r.path}
                </td>
                <td className="px-3 py-1.5 text-right text-[var(--orbit-muted)]">
                  {r.durationMs.toFixed(1)}ms
                </td>
                <td className="px-3 py-1.5 text-right text-[var(--orbit-muted)]">
                  {formatSize(r.bytes)}
                </td>
                <td className="px-3 py-1.5 text-right text-[var(--orbit-subtle)]">
                  {new Date(r.ts).toLocaleTimeString()}
                </td>
              </tr>
            ))}
            {rows.length === 0 && (
              <tr>
                <td colSpan={6} className="px-3 py-6 text-center text-[var(--orbit-muted)] italic">
                  No requests yet. Open the project in a browser to see traffic.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
