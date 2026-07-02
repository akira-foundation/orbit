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
  Info,
  Lock,
  LockOpen,
  Play,
  Plug,
  RotateCcw,
  Server,
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
import { ServicesPanel } from "../components/ServicesPanel";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { cn } from "../lib/cn";

const DETAIL_TABS: {
  id: "details" | "services" | "logs";
  label: string;
  subtitle: string;
  icon: typeof Info;
}[] = [
  { id: "details", label: "Details", subtitle: "Resolved project metadata", icon: Info },
  { id: "services", label: "Services", subtitle: "Local services this project uses", icon: Server },
  { id: "logs", label: "Logs", subtitle: "Live stdout / stderr from the runtime", icon: FileText },
];

export function ProjectDetail({ id }: { id: string }) {
  const { select, remove, showProjectMetrics, showProjectLogs } = useProjects();
  const { snapshot, logs, uptimeMs, start, stop, restart, clearLogs } =
    useRuntime(id);
  const [project, setProject] = useState<Project | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [confirmRemove, setConfirmRemove] = useState(false);
  const [tab, setTab] = useState<"details" | "services" | "logs">("details");
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
              <div className="mt-2 flex items-center gap-2">
                <button
                  onClick={() => api.openProject(project.id)}
                  className="group inline-flex items-center gap-1.5 text-[13px] font-mono text-[var(--orbit-accent-2)] hover:text-[var(--orbit-accent)] transition-colors"
                >
                  {project.secure ? "https://" : ""}{project.localDomain}
                  <ExternalLink className="size-3 opacity-60 group-hover:opacity-100" />
                </button>
                <SecureToggle
                  projectId={project.id}
                  secure={project.secure}
                  onChange={async (next) => {
                    setProject({ ...project, secure: next });
                    try {
                      await api.setProjectSecure(project.id, next);
                    } catch {
                      setProject({ ...project, secure: !next });
                    }
                  }}
                />
              </div>
            </div>

            <div className="flex items-center gap-1 shrink-0">
              <IconBtn
                icon={project.secure ? <Lock /> : <LockOpen />}
                onClick={async () => {
                  const next = !project.secure;
                  setProject({ ...project, secure: next });
                  try {
                    await api.setProjectSecure(project.id, next);
                  } catch {
                    setProject({ ...project, secure: !next });
                  }
                }}
                disabled={busy}
                title={project.secure ? "HTTPS only — click to disable" : "Force HTTPS"}
              />
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

        <div className="flex items-center gap-1 rounded-lg border border-[var(--orbit-border)] bg-white/[0.02] p-1 w-fit">
          {DETAIL_TABS.map((t) => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={cn(
                "h-7 px-3 rounded-md inline-flex items-center gap-1.5 text-[12px] transition",
                tab === t.id
                  ? "bg-white/[0.10] text-white"
                  : "text-white/70 hover:bg-white/[0.05]",
              )}
            >
              <t.icon className="size-3.5 shrink-0 text-[var(--orbit-muted)]" />
              {t.label}
            </button>
          ))}
        </div>

        <Card
          title={DETAIL_TABS.find((t) => t.id === tab)!.label}
          subtitle={DETAIL_TABS.find((t) => t.id === tab)!.subtitle}
        >
          {tab === "details" && (
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
          )}

          {tab === "services" && (
            <ServicesPanel projectId={id} isRunning={isRunning} />
          )}

          {tab === "logs" && (
            <>
              <LogsPanel logs={logs} onClear={clearLogs} />
              {snapshot?.error && (
                <p className="mt-2 text-[11px] text-rose-300 font-mono">
                  {snapshot.error}
                </p>
              )}
            </>
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

function SecureToggle({
  projectId,
  secure,
  onChange,
}: {
  projectId: string;
  secure: boolean;
  onChange: (next: boolean) => void;
}) {
  void projectId;
  return (
    <button
      onClick={() => onChange(!secure)}
      title={secure ? "HTTPS only — click to disable" : "Force HTTPS"}
      className={cn(
        "inline-flex items-center gap-1.5 h-6 px-2 rounded-md border text-[10.5px] font-medium tracking-wide transition-colors",
        secure
          ? "border-emerald-400/30 bg-emerald-400/10 text-emerald-300"
          : "border-white/10 bg-white/[0.03] text-[var(--orbit-muted)] hover:text-[var(--orbit-text)]",
      )}
    >
      {secure ? <Lock className="size-3" /> : <LockOpen className="size-3" />}
      {secure ? "HTTPS" : "HTTP"}
    </button>
  );
}

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
