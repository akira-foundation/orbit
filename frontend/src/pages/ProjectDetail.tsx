import { useEffect, useState } from "react";
import { useCopyToClipboard } from "usehooks-ts";
import { api } from "../api";
import type { Project } from "../types";
import { RuntimeStatusBadge } from "../components/RuntimeStatusBadge";
import { Button } from "../components/ui/button";
import {
  ExternalLink,
  Play,
  RotateCcw,
  Square,
  Trash2,
  Folder,
  Copy,
  Check,
} from "lucide-react";
import { useProjects } from "../store";
import { useRuntime, formatUptime } from "../hooks/useRuntime";
import { LogsPanel } from "../components/LogsPanel";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { cn } from "../lib/cn";

export function ProjectDetail({ id }: { id: string }) {
  const { select, remove } = useProjects();
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

  return (
    <div className="h-full overflow-auto scrollbar-thin">
      <div className="mx-auto w-full max-w-6xl px-10 py-8 space-y-8">

        <header className="flex items-center justify-between gap-8">
          <div className="min-w-0 space-y-1">
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-semibold tracking-tight truncate">
                {project.name}
              </h1>
              <RuntimeStatusBadge status={status} />
            </div>
            <button
              onClick={() => api.openProject(project.id)}
              className="inline-flex items-center gap-1.5 text-sm font-mono text-[var(--orbit-accent-2)] hover:text-[var(--orbit-accent)] transition-colors"
            >
              {project.localDomain}
              <ExternalLink className="size-3.5" />
            </button>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            {isRunning ? (
              <>
                <Button
                  variant="secondary"
                  onClick={() => wrap(restart)}
                  disabled={busy}
                  title="Restart"
                >
                  <RotateCcw />
                  Restart
                </Button>
                <Button
                  variant="secondary"
                  onClick={() => wrap(stop)}
                  disabled={busy}
                >
                  <Square />
                  Stop
                </Button>
              </>
            ) : (
              <Button onClick={() => wrap(start)} disabled={busy}>
                <Play />
                Start
              </Button>
            )}
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setConfirmRemove(true)}
              disabled={busy}
              title="Remove"
            >
              <Trash2 />
            </Button>
          </div>
        </header>


        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <Stat label="Framework" value={project.detectedFramework} />
          <Stat label="Package manager" value={project.packageManager} />
          <Stat
            label="Port"
            value={detectedPort ? String(detectedPort) : "—"}
            highlight={!!snapshot?.port}
          />
          <Stat
            label="Uptime"
            value={isRunning ? formatUptime(uptimeMs) : "—"}
            highlight={isRunning}
          />
        </div>


        <section className="space-y-3">
          <SectionTitle>Runtime</SectionTitle>
          <LogsPanel logs={logs} onClear={clearLogs} />
          {snapshot?.error && (
            <p className="text-xs text-rose-400 font-mono">{snapshot.error}</p>
          )}
        </section>


        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">

          <section className="space-y-4">
            <SectionTitle>Scripts</SectionTitle>
            {(project.scripts ?? []).length === 0 ? (
              <p className="text-sm text-[var(--orbit-muted)]">
                No scripts detected.
              </p>
            ) : (
              <div className="rounded-xl border border-[var(--orbit-border)] divide-y divide-[var(--orbit-border)] overflow-hidden">
                {(project.scripts ?? []).map((s) => (
                  <div
                    key={s.id}
                    className="group grid grid-cols-[6rem_1fr_auto] items-center gap-4 px-4 py-2.5 hover:bg-white/3 transition-colors"
                  >
                    <span className="text-xs font-semibold text-[var(--orbit-text)] truncate">
                      {s.name}
                    </span>
                    <code className="font-mono text-xs text-[var(--orbit-muted)] truncate">
                      {s.command}
                    </code>
                    <button
                      onClick={() => copy(`s-${s.id}`, s.command)}
                      className="opacity-0 group-hover:opacity-100 transition-opacity text-[var(--orbit-muted)] hover:text-[var(--orbit-text)]"
                      title="Copy"
                    >
                      {copiedKey === `s-${s.id}` ? (
                        <Check className="size-3.5 text-emerald-400" />
                      ) : (
                        <Copy className="size-3.5" />
                      )}
                    </button>
                  </div>
                ))}
              </div>
            )}
          </section>


          <section className="space-y-4">
            <SectionTitle>Details</SectionTitle>
            <dl className="rounded-xl border border-[var(--orbit-border)] divide-y divide-[var(--orbit-border)] overflow-hidden">
              <Row label="Dev command">
                <code className="font-mono text-xs">{project.devCommand}</code>
              </Row>
              <Row label="PID">
                <span className="font-mono text-xs">
                  {snapshot?.pid ? snapshot.pid : "—"}
                </span>
              </Row>
              <Row label="Path">
                <button
                  onClick={() => copy("path", project.path)}
                  className="group inline-flex items-center gap-1.5 font-mono text-xs text-[var(--orbit-text)] hover:text-[var(--orbit-accent-2)] transition-colors min-w-0 max-w-full"
                  title="Copy path"
                >
                  <Folder className="size-3 shrink-0 text-[var(--orbit-muted)]" />
                  <span className="truncate">{project.path}</span>
                  {copiedKey === "path" ? (
                    <Check className="size-3 shrink-0 text-emerald-400" />
                  ) : (
                    <Copy className="size-3 shrink-0 opacity-0 group-hover:opacity-100" />
                  )}
                </button>
              </Row>
              <Row label="Slug">
                <span className="font-mono text-xs">{project.slug}</span>
              </Row>
              <Row label="Created">
                <span className="text-xs text-[var(--orbit-muted)]">
                  {new Date(project.createdAt).toLocaleString()}
                </span>
              </Row>
            </dl>
          </section>
        </div>
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

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <h2 className="text-[11px] font-semibold uppercase tracking-widest text-[var(--orbit-subtle)]">
      {children}
    </h2>
  );
}

function Row({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="grid grid-cols-[7rem_1fr] items-center gap-4 px-4 py-2.5 min-w-0">
      <dt className="text-xs text-[var(--orbit-muted)]">{label}</dt>
      <dd className="min-w-0 text-right">{children}</dd>
    </div>
  );
}

function Stat({
  label,
  value,
  highlight,
}: {
  label: string;
  value: string;
  highlight?: boolean;
}) {
  return (
    <div
      className={cn(
        "rounded-xl border px-4 py-3 min-w-0 transition-colors",
        highlight
          ? "border-[var(--orbit-accent-2)]/40 bg-[var(--orbit-accent-2)]/8"
          : "border-[var(--orbit-border)] bg-white/2",
      )}
    >
      <div className="text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)]">
        {label}
      </div>
      <div className="mt-1.5 text-sm text-[var(--orbit-text)] truncate font-mono">
        {value}
      </div>
    </div>
  );
}
