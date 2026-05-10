import { useEffect, useMemo, useState } from "react";
import { useInterval, useCopyToClipboard } from "usehooks-ts";
import {
  ArrowLeft,
  Check,
  Copy,
  Eraser,
  Pause,
  Play,
  Search,
} from "lucide-react";
import Anser from "anser";
import { api } from "../api";
import { useProjects } from "../store";
import type { RuntimeLogLine } from "../types";
import { cn } from "../lib/cn";

const CURSOR_CTRL_RE = /\x1B\[[0-9;]*[A-HJKSTfsu]/g;
const STRAY_BRACKET_RE = /\[(?:\?25[lh]|2K|1G|0K|K|H|s|u)/g;

const RANGES: { label: string; ms: number }[] = [
  { label: "Last 15m", ms: 15 * 60 * 1000 },
  { label: "Last 1h", ms: 60 * 60 * 1000 },
  { label: "Last 6h", ms: 6 * 60 * 60 * 1000 },
  { label: "Last 24h", ms: 24 * 60 * 60 * 1000 },
  { label: "Last 7d", ms: 7 * 24 * 60 * 60 * 1000 },
];

type StreamFilter = "all" | "stdout" | "stderr" | "system";

export function ProjectLogsPage({ id }: { id: string }) {
  const { projects, select } = useProjects();
  const project = useMemo(
    () => projects.find((p) => p.id === id) ?? null,
    [projects, id],
  );

  const [rangeMs, setRangeMs] = useState(RANGES[1].ms);
  const [streamFilter, setStreamFilter] = useState<StreamFilter>("all");
  const [query, setQuery] = useState("");
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [logs, setLogs] = useState<RuntimeLogLine[]>([]);
  const [copied, setCopied] = useState(false);
  const [, copy] = useCopyToClipboard();

  const refresh = async () => {
    try {
      // ts is stored in nanoseconds; the backend interprets sinceTs in the same
      // unit. Math.round(ms * 1e6) gives a JS-safe ns value for our window.
      const sinceNs = Math.round((Date.now() - rangeMs) * 1_000_000);
      const data = await api.runtimeLogsHistory(id, sinceNs, 5000);
      setLogs(data);
    } catch {
      /* ignore */
    }
  };

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, rangeMs]);

  useInterval(refresh, autoRefresh ? 3000 : null);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return logs.filter((l) => {
      if (streamFilter !== "all" && l.stream !== streamFilter) return false;
      if (q && !l.text.toLowerCase().includes(q)) return false;
      return true;
    });
  }, [logs, streamFilter, query]);

  const onCopy = async () => {
    const text = filtered
      .map((l) => l.text.replace(CURSOR_CTRL_RE, "").replace(STRAY_BRACKET_RE, ""))
      .join("\n");
    if (await copy(text)) {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    }
  };

  if (!project) {
    return (
      <div className="h-full flex items-center justify-center">
        <p className="text-sm text-[var(--orbit-muted)]">Project not found.</p>
      </div>
    );
  }

  return (
    <div className="h-full overflow-hidden flex flex-col">
      <div className="px-10 pt-8 pb-4 space-y-3">
        <button
          onClick={() => select(id)}
          className="inline-flex items-center gap-1.5 text-[11px] text-[var(--orbit-muted)] hover:text-[var(--orbit-text)]"
        >
          <ArrowLeft className="size-3" />
          Back to {project.name}
        </button>
        <div className="flex items-baseline justify-between gap-4">
          <h1 className="text-2xl font-semibold tracking-tight">
            {project.name} · Logs
          </h1>
          <p className="text-[11px] text-[var(--orbit-muted)]">
            {filtered.length} of {logs.length} lines
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-1 rounded-md bg-white/5 border border-white/10 p-0.5">
            {RANGES.map((r) => (
              <button
                key={r.label}
                onClick={() => setRangeMs(r.ms)}
                className={cn(
                  "h-6 px-2.5 rounded text-[11px] font-medium transition-colors",
                  rangeMs === r.ms
                    ? "bg-white/10 text-white"
                    : "text-[var(--orbit-muted)] hover:text-white",
                )}
              >
                {r.label}
              </button>
            ))}
          </div>

          <div className="flex items-center gap-1 rounded-md bg-white/5 border border-white/10 p-0.5">
            {(["all", "stdout", "stderr", "system"] as StreamFilter[]).map(
              (s) => (
                <button
                  key={s}
                  onClick={() => setStreamFilter(s)}
                  className={cn(
                    "h-6 px-2.5 rounded text-[11px] font-medium transition-colors capitalize",
                    streamFilter === s
                      ? s === "stderr"
                        ? "bg-rose-400/15 text-rose-200"
                        : "bg-white/10 text-white"
                      : "text-[var(--orbit-muted)] hover:text-white",
                  )}
                >
                  {s}
                </button>
              ),
            )}
          </div>

          <label className="flex items-center gap-1.5 h-7 px-2.5 rounded-md bg-white/5 border border-white/10">
            <Search className="size-3.5 text-[var(--orbit-muted)]" />
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search lines…"
              className="bg-transparent text-[11px] outline-none w-48 placeholder:text-[var(--orbit-subtle)]"
            />
          </label>

          <div className="flex-1" />

          <button
            onClick={() => setAutoRefresh((v) => !v)}
            title={autoRefresh ? "Pause auto-refresh" : "Resume auto-refresh"}
            className={cn(
              "size-7 inline-flex items-center justify-center rounded-md border border-white/10 hover:bg-white/5",
              autoRefresh ? "text-emerald-400" : "text-[var(--orbit-muted)]",
            )}
          >
            {autoRefresh ? (
              <Pause className="size-3.5" />
            ) : (
              <Play className="size-3.5" />
            )}
          </button>
          <button
            onClick={onCopy}
            disabled={filtered.length === 0}
            title={copied ? "Copied" : "Copy filtered logs"}
            className={cn(
              "size-7 inline-flex items-center justify-center rounded-md border border-white/10 hover:bg-white/5 text-[var(--orbit-muted)]",
              copied && "text-emerald-400",
              filtered.length === 0 && "opacity-40 cursor-default",
            )}
          >
            {copied ? (
              <Check className="size-3.5" />
            ) : (
              <Copy className="size-3.5" />
            )}
          </button>
          <button
            onClick={() => setLogs([])}
            title="Clear view (does not delete persisted logs)"
            className="size-7 inline-flex items-center justify-center rounded-md border border-white/10 hover:bg-white/5 text-[var(--orbit-muted)]"
          >
            <Eraser className="size-3.5" />
          </button>
        </div>
      </div>

      <div className="flex-1 min-h-0 mx-10 mb-8 rounded-xl border border-[var(--orbit-border)] bg-black/40 overflow-auto scrollbar-thin font-mono text-[11px] leading-5 px-4 py-3">
        {filtered.length === 0 ? (
          <p className="text-[var(--orbit-muted)] italic">
            {logs.length === 0
              ? "No persisted logs in this range yet."
              : "No lines match the current filters."}
          </p>
        ) : (
          filtered.map((l, i) => (
            <div
              key={i}
              className={cn(
                "whitespace-pre-wrap break-all",
                l.stream === "stderr" && "text-rose-300",
                l.stream === "system" && "text-[var(--orbit-muted)] italic",
                l.stream === "stdout" && "text-[var(--orbit-text)]/85",
              )}
              dangerouslySetInnerHTML={{
                __html: Anser.ansiToHtml(
                  l.text
                    .replace(CURSOR_CTRL_RE, "")
                    .replace(STRAY_BRACKET_RE, ""),
                  { use_classes: false, json: false },
                ),
              }}
            />
          ))
        )}
      </div>
    </div>
  );
}
