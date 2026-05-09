import { useEffect, useState } from "react";
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
import { cn } from "../lib/cn";

export function ProjectDetail({ id }: { id: string }) {
  const { select, remove, start, stop } = useProjects();
  const [project, setProject] = useState<Project | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);

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
    await navigator.clipboard.writeText(text);
    setCopied(key);
    setTimeout(() => setCopied(null), 1200);
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

  const isRunning =
    project.status === "running" || project.status === "starting";

  const onStart = async () => {
    setBusy(true);
    try {
      await start(project.id);
      await reload();
    } finally {
      setBusy(false);
    }
  };

  const onStop = async () => {
    setBusy(true);
    try {
      await stop(project.id);
      await reload();
    } finally {
      setBusy(false);
    }
  };

  const onDelete = async () => {
    if (!confirm(`Remove "${project.name}" from Orbit?`)) return;
    await remove(project.id);
    select(null);
  };

  return (
    <div className="h-full overflow-auto scrollbar-thin">
      <div className="mx-auto w-full max-w-6xl px-10 py-8 space-y-10">
        {/* Hero */}
        <header className="flex items-center justify-between gap-8">
          <div className="min-w-0 flex items-center gap-4">
            <div className="min-w-0">
              <div className="flex items-center gap-3">
                <h1 className="text-2xl font-semibold tracking-tight truncate">
                  {project.name}
                </h1>
                <RuntimeStatusBadge status={project.status} />
              </div>
              <a
                href={`https://${project.localDomain}`}
                target="_blank"
                rel="noreferrer"
                className="mt-1 inline-flex items-center gap-1.5 text-sm font-mono text-[var(--orbit-accent-2)] hover:text-[var(--orbit-accent)] transition-colors"
              >
                {project.localDomain}
                <ExternalLink className="size-3.5" />
              </a>
            </div>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            {isRunning ? (
              <Button variant="secondary" onClick={onStop} disabled={busy}>
                <Square />
                Stop
              </Button>
            ) : (
              <Button onClick={onStart} disabled={busy}>
                <Play />
                Start
              </Button>
            )}
            <Button
              variant="ghost"
              size="icon"
              onClick={onDelete}
              disabled={busy}
              title="Remove"
            >
              <Trash2 />
            </Button>
          </div>
        </header>

        {/* Stats */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <Stat label="Framework" value={project.detectedFramework} />
          <Stat label="Package manager" value={project.packageManager} />
          <Stat
            label="Dev port"
            value={project.devPort ? String(project.devPort) : "—"}
          />
          <Stat label="Dev command" value={project.devCommand} mono />
        </div>

        {/* Body */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* Scripts */}
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
                      {copied === `s-${s.id}` ? (
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

          {/* Details */}
          <section className="space-y-4">
            <SectionTitle>Details</SectionTitle>
            <dl className="rounded-xl border border-[var(--orbit-border)] divide-y divide-[var(--orbit-border)] overflow-hidden">
              <Row label="Path">
                <button
                  onClick={() => copy("path", project.path)}
                  className="group inline-flex items-center gap-1.5 font-mono text-xs text-[var(--orbit-text)] hover:text-[var(--orbit-accent-2)] transition-colors min-w-0 max-w-full"
                  title="Copy path"
                >
                  <Folder className="size-3 shrink-0 text-[var(--orbit-muted)]" />
                  <span className="truncate">{project.path}</span>
                  {copied === "path" ? (
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
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="rounded-xl border border-[var(--orbit-border)] bg-white/2 px-4 py-3 min-w-0">
      <div className="text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)]">
        {label}
      </div>
      <div
        className={cn(
          "mt-1.5 text-sm text-[var(--orbit-text)] truncate",
          mono && "font-mono text-xs",
        )}
      >
        {value}
      </div>
    </div>
  );
}
