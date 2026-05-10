import { useEffect, useMemo, useRef, useState } from "react";
import Anser from "anser";
import { useCopyToClipboard } from "usehooks-ts";
import { Check, Copy, Eraser, Pause, Play } from "lucide-react";
import type { RuntimeLogLine } from "../types";
import { cn } from "../lib/cn";

const CURSOR_CTRL_RE = /\x1B\[[0-9;]*[A-HJKSTfsu]/g;
const STRAY_BRACKET_RE = /\[(?:\?25[lh]|2K|1G|0K|K|H|s|u)/g;

function renderAnsi(text: string): string {
  const cleaned = text.replace(CURSOR_CTRL_RE, "").replace(STRAY_BRACKET_RE, "");
  return Anser.ansiToHtml(cleaned, { use_classes: false, json: false });
}

interface Props {
  logs: RuntimeLogLine[];
  onClear: () => void;
}

export function LogsPanel({ logs, onClear }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const [copied, setCopied] = useState(false);
  const [, copy] = useCopyToClipboard();

  const onCopy = async () => {
    const text = logs
      .map((l) => l.text.replace(CURSOR_CTRL_RE, "").replace(STRAY_BRACKET_RE, ""))
      .join("\n");
    const ok = await copy(text);
    if (ok) {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    }
  };

  const rendered = useMemo(
    () => logs.map((l) => ({ ...l, html: renderAnsi(l.text) })),
    [logs],
  );

  useEffect(() => {
    if (!autoScroll) return;
    const el = ref.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [logs, autoScroll]);

  const onScroll = () => {
    const el = ref.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
    setAutoScroll(atBottom);
  };

  return (
    <div className="rounded-xl border border-[var(--orbit-border)] bg-black/40 overflow-hidden flex flex-col h-80">
      <div className="flex items-center justify-between px-3 h-8 border-b border-[var(--orbit-border)] bg-white/2">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--orbit-subtle)]">
          Logs
          <span className="ml-2 text-[var(--orbit-muted)] normal-case tracking-normal font-normal">
            {logs.length} lines
          </span>
        </span>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setAutoScroll((v) => !v)}
            className={cn(
              "size-6 inline-flex items-center justify-center rounded text-[var(--orbit-muted)] hover:text-[var(--orbit-text)] hover:bg-white/5 transition-colors",
              autoScroll && "text-emerald-400 hover:text-emerald-300",
            )}
            title={autoScroll ? "Pause auto-scroll" : "Resume auto-scroll"}
          >
            {autoScroll ? <Pause className="size-3.5" /> : <Play className="size-3.5" />}
          </button>
          <button
            onClick={onCopy}
            disabled={logs.length === 0}
            className={cn(
              "size-6 inline-flex items-center justify-center rounded text-[var(--orbit-muted)] hover:text-[var(--orbit-text)] hover:bg-white/5 transition-colors",
              copied && "text-emerald-400 hover:text-emerald-300",
              logs.length === 0 && "opacity-40 cursor-default",
            )}
            title={copied ? "Copied" : "Copy logs"}
          >
            {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
          </button>
          <button
            onClick={onClear}
            className="size-6 inline-flex items-center justify-center rounded text-[var(--orbit-muted)] hover:text-[var(--orbit-text)] hover:bg-white/5 transition-colors"
            title="Clear logs"
          >
            <Eraser className="size-3.5" />
          </button>
        </div>
      </div>
      <div
        ref={ref}
        onScroll={onScroll}
        className="flex-1 overflow-auto scrollbar-thin font-mono text-[11px] leading-5 px-3 py-2"
      >
        {logs.length === 0 ? (
          <p className="text-[var(--orbit-muted)] italic">No logs yet.</p>
        ) : (
          rendered.map((l, i) => (
            <div
              key={i}
              className={cn(
                "whitespace-pre-wrap break-all",
                l.stream === "stderr" && "text-rose-300",
                l.stream === "system" && "text-[var(--orbit-muted)] italic",
                l.stream === "stdout" && "text-[var(--orbit-text)]/85",
              )}
              dangerouslySetInnerHTML={{ __html: l.html }}
            />
          ))
        )}
      </div>
    </div>
  );
}
