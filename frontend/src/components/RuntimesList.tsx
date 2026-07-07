import { useEffect, useState } from "react";
import { api } from "../api";
import type {
  NodeVersionInfo,
  PHPVersionInfo,
  RuntimesConfig,
  SystemRuntimeStatus,
} from "../types";
import { ConfirmDialog } from "./ConfirmDialog";
import { ToggleRow } from "./SettingsSections";
import { RuntimeVersionRow, RuntimeVersionPicker } from "./RuntimeRows";

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
    <div className="rounded-lg border border-white/10 bg-white/[0.03]">
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
      {versions.length > 1 ? (
        <RuntimeVersionPicker
          label="Node.js"
          versions={versions}
          busy={busy}
          onInstall={install}
          onRemove={setRemoveFor}
        />
      ) : (
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
      )}

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
    <div className="rounded-lg border border-white/10 bg-white/[0.03]">
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
      {versions.length > 1 ? (
        <RuntimeVersionPicker
          label="PHP"
          versions={versions}
          busy={busy}
          onInstall={install}
          onRemove={setRemoveFor}
        />
      ) : (
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
      )}

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
