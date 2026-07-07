import { useEffect, useState } from "react";
import type { SystemRuntimeStatus } from "../types";
import { ConfirmDialog } from "./ConfirmDialog";
import { ToggleRow } from "./SettingsSections";
import {
  RuntimeVersionRow,
  RuntimeVersionPicker,
  type VersionInfo,
} from "./RuntimeRows";

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

export function RuntimeGroup<T extends VersionInfo>({
  label,
  description,
  system,
  preferChecked,
  onPreferChange,
  list,
  install,
  remove,
}: {
  label: string;
  description: string;
  system: SystemRuntimeStatus | null;
  preferChecked: boolean;
  onPreferChange: (next: boolean) => void;
  list: () => Promise<T[]>;
  install: (version: string) => Promise<void>;
  remove: (version: string) => Promise<void>;
}) {
  const [versions, setVersions] = useState<T[]>([]);
  const [busy, setBusy] = useState<string | null>(null);
  const [removeFor, setRemoveFor] = useState<T | null>(null);

  async function refresh() {
    setVersions(await list());
  }

  useEffect(() => {
    refresh();
  }, []);

  async function onInstall(v: T) {
    setBusy(v.version);
    try {
      await install(v.version);
      await refresh();
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="rounded-lg border border-white/10 bg-white/[0.03]">
      <div className="px-4 pt-3 pb-2 border-b border-white/[0.06] space-y-0.5">
        <p className="text-[13px] font-medium">{label}</p>
        <p className="text-[11px] text-[var(--orbit-muted)] leading-relaxed">
          {description}
        </p>
      </div>
      <SystemPreferenceToggle
        label={label}
        system={system}
        checked={preferChecked}
        onChange={onPreferChange}
      />
      {preferChecked ? null : versions.length > 1 ? (
        <RuntimeVersionPicker
          label={label}
          versions={versions}
          busy={busy}
          onInstall={onInstall}
          onRemove={setRemoveFor}
        />
      ) : (
        <div className="divide-y divide-white/[0.04]">
          {versions.map((v) => (
            <RuntimeVersionRow
              key={v.id}
              label={label}
              v={v}
              busy={busy === v.version}
              onInstall={onInstall}
              onRemove={setRemoveFor}
            />
          ))}
        </div>
      )}

      <ConfirmDialog
        open={!!removeFor}
        onOpenChange={(o) => !o && setRemoveFor(null)}
        title={`Remove ${label} ${removeFor?.version ?? ""}?`}
        description="Deletes the downloaded binary. Projects pinned to this version will re-download it automatically on next start."
        confirmLabel="Remove"
        destructive
        onConfirm={async () => {
          if (removeFor) {
            await remove(removeFor.version);
            await refresh();
          }
        }}
      />
    </div>
  );
}
