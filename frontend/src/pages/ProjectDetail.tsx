import { useEffect, useState } from "react";
import { useCopyToClipboard } from "usehooks-ts";
import { api } from "../api";
import type { Project } from "../types";
import { RuntimeStatusBadge } from "../components/RuntimeStatusBadge";
import { Button } from "../components/ui/button";
import {
  BarChart3,
  ExternalLink,
  FileText,
  FolderOpen,
  Play,
  Plug,
  RotateCcw,
  Square,
  Terminal,
  Timer,
  Trash2,
  Copy,
  Check,
} from "lucide-react";
import { useProjects } from "../store";
import { useRuntime, formatUptime } from "../hooks/useRuntime";
import { LogsPanel } from "../components/LogsPanel";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { cn } from "../lib/cn";

export function ProjectDetail({ id }: { id: string }) {
  const { select, remove, showProjectMetrics, showProjectLogs } = useProjects();
  const { snapshot, logs, uptimeMs, start, stop, restart, clearLogs } =
    useRuntime(id);
  const [project, setProject] = useState<Project | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [confirmRemove, setConfirmRemove] = useState(false);
  const [, copyToClipboard] = useCopyToClipboard();

  const reload = async () => {
    setError(null);
    try {
      setProject(await api.getProject(id));
    } catch (e: any) {
      setError(e?.message ?? String(e));
    }
  };

  useEffect(() => {
    setProject(null);
    setError(null);
    reload();
  }, [id]);

  const copy = async (key: string, text: string) => {
    await copyToClipboard(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 1200);
  };

  if (error) {
    return (
      <div className="h-full flex items-center justify-center p-8">
        <div className="rounded-2xl border border-[var(--orbit-border)] bg-white/3 px-6 py-5 max-w-md text-center space-y-3">
          <p className="text-sm font-medium text-destructive">
            Failed to load project
          </p>
          <p className="text-xs text-[var(--orbit-muted)]">{error}</p>
          <Button variant="ghost" size="sm" onClick={reload}>
            <RotateCcw className="size-3.5" />
            Retry
          </Button>
        </div>
      </div>
    );
  }

  if (!project) {
    return (
      <div className="h-full flex items-center justify-center">
        <p className="text-sm text-[var(--orbit-muted)]">Loading…</p>
      </div>
    );
  }

  const status = snapshot?.status ?? project.status;
  const isRunning = status === "running" || status === "starting";
  const detectedPort = snapshot?.port || project.devPort;

  const wrap = async (fn: () => Promise<void>) => {
    setBusy(true);
    try {
      await fn();
    } finally {
      setBusy(false);
    }
  };

  const onDelete = async () => {
    if (isRunning) await stop();
    await remove(project.id);
    select(null);
  };

  // Status accent — dot color + soft glow tinted by the running state.
  // Drives the hero's left ornament and the ambient backdrop wash.
  const accent = {
    running:   { dot: "bg-emerald-400", glow: "rgba(52,211,153,0.18)" },
    starting:  { dot: "bg-cyan-400 animate-pulse", glow: "rgba(34,211,238,0.18)" },
    idle:      { dot: "bg-sky-400",     glow: "rgba(96,165,250,0.14)" },
    suspended: { dot: "bg-violet-400",  glow: "rgba(167,139,250,0.16)" },
    stopped:   { dot: "bg-zinc-500",    glow: "rgba(255,255,255,0.04)" },
    error:     { dot: "bg-rose-400",    glow: "rgba(251,113,133,0.18)" },
  }[status];

  return (
    <div className="h-full overflow-auto scrollbar-thin">
      <div className="mx-auto w-full max-w-5xl px-8 py-10 space-y-5">

        {/* HERO — single tight card, ambient glow keyed by status. */}
        <header
          className="relative rounded-2xl border border-[var(--orbit-border)] bg-white/2 px-6 py-6 overflow-hidden"
          style={{
            backgroundImage: `radial-gradient(70% 120% at 0% 0%, ${accent.glow} 0%, transparent 60%)`,
          }}
        >
          <div className="flex items-start justify-between gap-6">
            <div className="min-w-0">
              <div className="flex items-center gap-3">
                <h1 className="text-[24px] font-semibold tracking-tight leading-none truncate">
                  {project.name}
                </h1>
                <RuntimeStatusBadge status={status} />
              </div>
              <button
                onClick={() => api.openProject(project.id)}
                className="mt-2 group inline-flex items-center gap-1.5 text-[13px] font-mono text-[var(--orbit-accent-2)] hover:text-[var(--orbit-accent)] transition-colors"
              >
                {project.localDomain}
                <ExternalLink className="size-3 opacity-60 group-hover:opacity-100" />
              </button>
            </div>

            <div className="flex items-center gap-1 shrink-0">
              <IconBtn icon={<FileText />} onClick={() => showProjectLogs(project.id)} disabled={busy} title="Logs" />
              <IconBtn icon={<BarChart3 />} onClick={() => showProjectMetrics(project.id)} disabled={busy} title="Metrics" />
              <IconBtn icon={<Trash2 />} onClick={() => setConfirmRemove(true)} disabled={busy} title="Remove" />
              <span className="mx-1.5 h-5 w-px bg-white/[0.08]" />
              {isRunning ? (
                <>
                  <IconBtn icon={<RotateCcw />} onClick={() => wrap(restart)} disabled={busy} title="Restart" />
                  <Button
                    size="sm"
                    onClick={() => wrap(stop)}
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
                  onClick={() => wrap(start)}
                  disabled={busy}
                  className="bg-emerald-400/15 text-emerald-200 hover:bg-emerald-400/25 border border-emerald-400/30"
                >
                  <Play />
                  Start
                </Button>
              )}
            </div>
          </div>

          {/* Compact meta row inside the hero. Single connected strip
              divided by faint borders — reads as one unit, not 4 tiles. */}
          <div className="mt-5 grid grid-cols-2 md:grid-cols-4 divide-x divide-white/[0.06] border-t border-white/[0.06] -mx-6 -mb-5">
            <Cell icon={<Terminal className="size-3.5" />} label="Framework" value={project.detectedFramework} />
            <Cell icon={<Plug className="size-3.5" />} label="Package" value={project.packageManager} />
            <Cell
              icon={<Plug className="size-3.5" />}
              label="Port"
              value={detectedPort ? String(detectedPort) : "—"}
              accent={!!snapshot?.port}
            />
            <Cell
              icon={<Timer className="size-3.5" />}
              label="Uptime"
              value={isRunning ? formatUptime(uptimeMs) : "—"}
              ok={isRunning}
            />
          </div>
        </header>

        <Card title="Details" subtitle="Resolved project metadata">
          <dl className="divide-y divide-white/[0.04]">
            <Detail label="Dev command">
              <code className="font-mono text-[11.5px]">
                {project.devCommand}
              </code>
            </Detail>
            <Detail label="PID">
              <span className="font-mono text-[11.5px]">
                {snapshot?.pid ? snapshot.pid : "—"}
              </span>
            </Detail>
            <Detail label="Path">
              <div className="group inline-flex items-center gap-1.5 min-w-0 max-w-full justify-end">
                <button
                  onClick={() => copy("path", project.path)}
                  className="font-mono text-[11.5px] text-[var(--orbit-text)] hover:text-[var(--orbit-accent-2)] transition-colors truncate text-right"
                  title="Copy path"
                >
                  {project.path}
                </button>
                <button
                  onClick={() => api.revealInFinder(project.path)}
                  className="text-[var(--orbit-muted)] hover:text-[var(--orbit-text)]"
                  title="Reveal in Finder"
                >
                  <FolderOpen className="size-3.5" />
                </button>
              </div>
            </Detail>
            <Detail label="Slug">
              <code className="font-mono text-[11.5px]">{project.slug}</code>
            </Detail>
            <Detail label="Created">
              <span className="text-[11.5px] text-[var(--orbit-muted)]">
                {new Date(project.createdAt).toLocaleString()}
              </span>
            </Detail>
          </dl>
        </Card>

        <Card
          title="Logs"
          subtitle="Live stdout / stderr from the runtime"
        >
          <LogsPanel logs={logs} onClear={clearLogs} />
          {snapshot?.error && (
            <p className="mt-2 text-[11px] text-rose-300 font-mono">
              {snapshot.error}
            </p>
          )}
        </Card>
      </div>

      <ConfirmDialog
        open={confirmRemove}
        onOpenChange={setConfirmRemove}
        title={`Remove "${project.name}"?`}
        description={
          <>
            Project will be removed from Orbit. Source files on disk stay
            untouched.
            {isRunning && (
              <span className="block mt-1 text-rose-300">
                Runtime is active and will be stopped first.
              </span>
            )}
          </>
        }
        confirmLabel="Remove"
        destructive
        onConfirm={onDelete}
      />
    </div>
  );
}

// Cell — a single hero meta cell (Framework / Package / Port / Uptime).
// No label uppercase tracking — just sentence case. Value mono-styled.
function Cell({
  icon,
  label,
  value,
  accent,
  ok,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  accent?: boolean;
  ok?: boolean;
}) {
  const valCls = ok
    ? "text-emerald-300"
    : accent
      ? "text-[var(--orbit-accent-2)]"
      : "text-[var(--orbit-text)]";
  return (
    <div className="px-6 py-4 min-w-0">
      <div className="flex items-center gap-1.5 text-[11px] text-[var(--orbit-muted)]">
        <span className="text-[var(--orbit-subtle)]">{icon}</span>
        {label}
      </div>
      <div className={cn("mt-1 text-[14px] font-mono truncate", valCls)}>
        {value}
      </div>
    </div>
  );
}

// Card — same visual language as the Metrics page ChartCard. Rounded-2xl,
// subtle border + bg, p-5, header with title + optional subtitle muted.
function Card({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-6 space-y-5">
      <header className="space-y-0.5">
        <h2 className="text-[15px] font-semibold tracking-tight">{title}</h2>
        {subtitle && (
          <p className="text-[11.5px] text-[var(--orbit-muted)] leading-relaxed">
            {subtitle}
          </p>
        )}
      </header>
      {children}
    </section>
  );
}

// Detail — clean dt/dd row inside the Details column.
function Detail({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="grid grid-cols-[7rem_1fr] items-center gap-3 py-3 first:pt-0 last:pb-0">
      <dt className="text-[12px] text-[var(--orbit-muted)]">{label}</dt>
      <dd className="min-w-0 text-right">{children}</dd>
    </div>
  );
}

// IconBtn — small ghost icon button used in the hero toolbar.
function IconBtn({
  icon,
  onClick,
  disabled,
  title,
}: {
  icon: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  title: string;
}) {
  return (
    <Button variant="ghost" size="icon" onClick={onClick} disabled={disabled} title={title}>
      {icon}
    </Button>
  );
}
