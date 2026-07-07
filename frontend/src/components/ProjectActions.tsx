import { BarChart3, FileText, Lock, LockOpen, Play, RotateCcw, Square, Trash2 } from "lucide-react";
import { Button } from "./ui/button";
import { IconBtn } from "./ProjectDetailParts";

export function ProjectActions({
  secure,
  busy,
  isRunning,
  onToggleSecure,
  onLogs,
  onMetrics,
  onRemove,
  onRestart,
  onStop,
  onStart,
}: {
  secure: boolean;
  busy: boolean;
  isRunning: boolean;
  onToggleSecure: () => void;
  onLogs: () => void;
  onMetrics: () => void;
  onRemove: () => void;
  onRestart: () => void;
  onStop: () => void;
  onStart: () => void;
}) {
  return (
    <div className="flex items-center gap-1 shrink-0">
      <IconBtn
        icon={secure ? <Lock /> : <LockOpen />}
        onClick={onToggleSecure}
        disabled={busy}
        title={secure ? "HTTPS only — click to disable" : "Force HTTPS"}
      />
      <IconBtn icon={<FileText />} onClick={onLogs} disabled={busy} title="Logs" />
      <IconBtn icon={<BarChart3 />} onClick={onMetrics} disabled={busy} title="Metrics" />
      <IconBtn icon={<Trash2 />} onClick={onRemove} disabled={busy} title="Remove" />
      <span className="mx-1.5 h-5 w-px bg-white/[0.08]" />
      {isRunning ? (
        <>
          <IconBtn icon={<RotateCcw />} onClick={onRestart} disabled={busy} title="Restart" />
          <Button
            size="sm"
            onClick={onStop}
            disabled={busy}
            className="bg-rose-400/15 text-rose-200 hover:bg-rose-400/25 border border-rose-400/30"
          >
            <Square />
            Stop
          </Button>
        </>
      ) : (
        <Button
          size="sm"
          onClick={onStart}
          disabled={busy}
          className="bg-emerald-400/15 text-emerald-200 hover:bg-emerald-400/25 border border-emerald-400/30"
        >
          <Play />
          Start
        </Button>
      )}
    </div>
  );
}
