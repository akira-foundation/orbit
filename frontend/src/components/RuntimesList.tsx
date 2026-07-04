import { useEffect, useState } from "react";
import { Download, FolderOpen, Loader2, Trash2 } from "lucide-react";
import { api } from "../api";
import type {
  NodeVersionInfo,
  PHPVersionInfo,
  RuntimesConfig,
  SystemRuntimeStatus,
} from "../types";
import { cn } from "../lib/cn";
import { Button } from "./ui/button";
import { ConfirmDialog } from "./ConfirmDialog";
import { ToggleRow } from "./SettingsSections";

type VersionInfo = NodeVersionInfo | PHPVersionInfo;

function formatBytes(n: number): string {
  if (n <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.min(Math.floor(Math.log(n) / Math.log(1024)), units.length - 1);
  return `${(n / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function RuntimeVersionRow<T extends VersionInfo>({
  label,
  v,
  busy,
  onInstall,
  onRemove,
}: {
  label: string;
  v: T;
  busy: boolean;
  onInstall: (v: T) => void;
  onRemove: (v: T) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-4 px-4 py-3">
      <div className="flex items-start gap-3 min-w-0">
        <span
          className={cn(
            "mt-1.5 size-2 rounded-full shrink-0",
            v.installed ? "bg-emerald-400 shadow-[0_0_8px_#34d399]" : "bg-white/20",
          )}
        />
        <div className="min-w-0 space-y-0.5">
          <p className="text-[13px] font-medium">{label} {v.version}</p>
          <p className="text-[11px] text-[var(--orbit-subtle)]">
            {v.version} ·{" "}
            {v.installed ? formatBytes(v.diskBytes) : "not downloaded"}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-1.5 shrink-0">
        {v.installed ? (
          <>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => api.revealInFinder(v.path)}
              title="Reveal in Finder"
            >
              <FolderOpen className="size-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="text-rose-300 hover:bg-rose-500/10 hover:text-rose-200"
              disabled={busy}
              onClick={() => onRemove(v)}
              title="Remove"
            >
              <Trash2 className="size-3.5" />
            </Button>
          </>
        ) : (
          <Button
            variant="outline"
            size="sm"
            disabled={busy}
            onClick={() => onInstall(v)}
            title={busy ? "Downloading…" : "Download"}
          >
            {busy ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <Download className="size-3.5" />
            )}
          </Button>
        )}
      </div>
    </div>
  );
}

function SystemPreferenceToggle({
  label,
  system,
  checked,
  onChange,
}: {
  label: string;
  system: SystemRuntimeStatus | null;
  checked: boolean;
  onChange: (next: boolean) => void;
}) {
  if (!system?.available) return null;
  return (
    <div className="px-4 py-3 border-b border-white/[0.06]">
      <ToggleRow
        label={`Use system ${label} when available`}
        description={`Detected ${label} ${system.version} on this machine. Skips the bundled download and runs projects with it instead.`}
        checked={checked}
        onChange={onChange}
      />
    </div>
  );
}

function NodeRuntimeGroup({
  cfg,
  onSaveCfg,
  systemNode,
}: {
  cfg: RuntimesConfig | null;
  onSaveCfg: (next: RuntimesConfig) => void;
  systemNode: SystemRuntimeStatus | null;
}) {
  const [versions, setVersions] = useState<NodeVersionInfo[]>([]);
  const [busy, setBusy] = useState<string | null>(null);
  const [removeFor, setRemoveFor] = useState<NodeVersionInfo | null>(null);

  async function refresh() {
    setVersions(await api.listNodeVersions());
  }

  useEffect(() => {
    refresh();
  }, []);

  async function install(v: NodeVersionInfo) {
    setBusy(v.version);
    try {
      await api.installNodeVersion(v.version);
      await refresh();
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="rounded-lg border border-white/10 bg-white/[0.03] overflow-hidden">
      <div className="px-4 pt-3 pb-2 border-b border-white/[0.06] space-y-0.5">
        <p className="text-[13px] font-medium">Node.js</p>
        <p className="text-[11px] text-[var(--orbit-muted)] leading-relaxed">
          Bundled versions used to run project dev commands. Resolved per
          project from .nvmrc, .node-version, or package.json engines.node.
        </p>
      </div>
      {cfg && (
        <SystemPreferenceToggle
          label="Node.js"
          system={systemNode}
          checked={cfg.preferSystemNode}
          onChange={(v) => onSaveCfg({ ...cfg, preferSystemNode: v })}
        />
      )}
      <div className="divide-y divide-white/[0.04]">
        {versions.map((v) => (
          <RuntimeVersionRow
            key={v.id}
            label="Node.js"
            v={v}
            busy={busy === v.version}
            onInstall={install}
            onRemove={setRemoveFor}
          />
        ))}
      </div>

      <ConfirmDialog
        open={!!removeFor}
        onOpenChange={(o) => !o && setRemoveFor(null)}
        title={`Remove Node.js ${removeFor?.version ?? ""}?`}
        description="Deletes the downloaded binary. Projects pinned to this version will re-download it automatically on next start."
        confirmLabel="Remove"
        destructive
        onConfirm={async () => {
          if (removeFor) {
            await api.removeNodeVersion(removeFor.version);
            await refresh();
          }
        }}
      />
    </div>
  );
}

function PHPRuntimeGroup({
  cfg,
  onSaveCfg,
  systemPHP,
}: {
  cfg: RuntimesConfig | null;
  onSaveCfg: (next: RuntimesConfig) => void;
  systemPHP: SystemRuntimeStatus | null;
}) {
  const [versions, setVersions] = useState<PHPVersionInfo[]>([]);
  const [busy, setBusy] = useState<string | null>(null);
  const [removeFor, setRemoveFor] = useState<PHPVersionInfo | null>(null);

  async function refresh() {
    setVersions(await api.listPHPVersions());
  }

  useEffect(() => {
    refresh();
  }, []);

  async function install(v: PHPVersionInfo) {
    setBusy(v.version);
    try {
      await api.installPHPVersion(v.version);
      await refresh();
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="rounded-lg border border-white/10 bg-white/[0.03] overflow-hidden">
      <div className="px-4 pt-3 pb-2 border-b border-white/[0.06] space-y-0.5">
        <p className="text-[13px] font-medium">PHP</p>
        <p className="text-[11px] text-[var(--orbit-muted)] leading-relaxed">
          Bundled versions with php-fpm, used to run Laravel projects.
          Resolved per project from composer.json require.php.
        </p>
      </div>
      {cfg && (
        <SystemPreferenceToggle
          label="PHP"
          system={systemPHP}
          checked={cfg.preferSystemPhp}
          onChange={(v) => onSaveCfg({ ...cfg, preferSystemPhp: v })}
        />
      )}
      <div className="divide-y divide-white/[0.04]">
        {versions.map((v) => (
          <RuntimeVersionRow
            key={v.id}
            label="PHP"
            v={v}
            busy={busy === v.version}
            onInstall={install}
            onRemove={setRemoveFor}
          />
        ))}
      </div>

      <ConfirmDialog
        open={!!removeFor}
        onOpenChange={(o) => !o && setRemoveFor(null)}
        title={`Remove PHP ${removeFor?.version ?? ""}?`}
        description="Deletes the downloaded binary. Projects pinned to this version will re-download it automatically on next start."
        confirmLabel="Remove"
        destructive
        onConfirm={async () => {
          if (removeFor) {
            await api.removePHPVersion(removeFor.version);
            await refresh();
          }
        }}
      />
    </div>
  );
}

export function RuntimesList() {
  const [cfg, setCfg] = useState<RuntimesConfig | null>(null);
  const [systemNode, setSystemNode] = useState<SystemRuntimeStatus | null>(null);
  const [systemPHP, setSystemPHP] = useState<SystemRuntimeStatus | null>(null);

  useEffect(() => {
    api.runtimesConfig().then(setCfg);
    api.detectSystemNode().then(setSystemNode);
    api.detectSystemPHP().then(setSystemPHP);
  }, []);

  async function saveCfg(next: RuntimesConfig) {
    setCfg(next);
    await api.saveRuntimesConfig(next);
  }

  return (
    <div className="space-y-2.5">
      <NodeRuntimeGroup cfg={cfg} onSaveCfg={saveCfg} systemNode={systemNode} />
      <PHPRuntimeGroup cfg={cfg} onSaveCfg={saveCfg} systemPHP={systemPHP} />
    </div>
  );
}
