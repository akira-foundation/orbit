import { useEffect, useState } from "react";
import { useCopyToClipboard } from "usehooks-ts";
import { api } from "../api";
import type { Project } from "../types";
import { RuntimeStatusBadge } from "../components/RuntimeStatusBadge";
import { Button } from "../components/ui/button";
import {
  ExternalLink,
  FileText,
  Info,
  KeyRound,
  Plug,
  RotateCcw,
  Server,
  Terminal,
  Timer,
} from "lucide-react";
import { useProjects } from "../store";
import { useRuntime, formatUptime } from "../hooks/useRuntime";
import { LogsPanel } from "../components/LogsPanel";
import { ServicesPanel } from "../components/ServicesPanel";
import { ProjectTerminal } from "../components/ProjectTerminal";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { EnvEditorPanel } from "../components/EnvEditorPanel";
import { Card, Cell, SecureToggle } from "../components/ProjectDetailParts";
import { ProjectActions } from "../components/ProjectActions";
import { ProjectDetailsTab } from "../components/ProjectDetailsTab";
import { SharePanel } from "../components/SharePanel";
import { cn } from "../lib/cn";

type DetailTab = "details" | "services" | "env" | "logs" | "terminal";

const DETAIL_TABS: {
  id: DetailTab;
  label: string;
  subtitle: string;
  icon: typeof Info;
}[] = [
  { id: "details", label: "Details", subtitle: "Resolved project metadata", icon: Info },
  { id: "services", label: "Services", subtitle: "Local services this project uses", icon: Server },
  { id: "env", label: "Env", subtitle: "Project .env variables", icon: KeyRound },
  { id: "logs", label: "Logs", subtitle: "Live stdout / stderr from the runtime", icon: FileText },
  { id: "terminal", label: "Terminal", subtitle: "Interactive shell at the project root", icon: Terminal },
];

export function ProjectDetail({ id }: { id: string }) {
  const { select, remove, showProjectMetrics, showProjectLogs } = useProjects();
  const { snapshot, logs, uptimeMs, start, stop, restart, clearLogs } =
    useRuntime(id);
  const [project, setProject] = useState<Project | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [confirmRemove, setConfirmRemove] = useState(false);
  const [shareOpen, setShareOpen] = useState(false);
  const [tab, setTab] = useState<DetailTab>("details");
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

  const copy = async (_key: string, text: string) => {
    await copyToClipboard(text);
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

            <ProjectActions
              secure={project.secure}
              busy={busy}
              isRunning={isRunning}
              onToggleSecure={async () => {
                const next = !project.secure;
                setProject({ ...project, secure: next });
                try {
                  await api.setProjectSecure(project.id, next);
                } catch {
                  setProject({ ...project, secure: !next });
                }
              }}
              onShare={() => setShareOpen(true)}
              onLogs={() => showProjectLogs(project.id)}
              onMetrics={() => showProjectMetrics(project.id)}
              onRemove={() => setConfirmRemove(true)}
              onRestart={() => wrap(restart)}
              onStop={() => wrap(stop)}
              onStart={() => wrap(start)}
            />
          </div>

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
            <ProjectDetailsTab
              project={project}
              snapshot={snapshot}
              onCopy={copy}
            />
          )}

          {tab === "services" && (
            <ServicesPanel projectId={id} isRunning={isRunning} />
          )}

          {tab === "env" && <EnvEditorPanel projectId={id} />}

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

          {tab === "terminal" && <ProjectTerminal projectId={id} />}
        </Card>
      </div>

      <SharePanel
        projectId={id}
        open={shareOpen}
        onClose={() => setShareOpen(false)}
      />

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
