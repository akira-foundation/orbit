import { useState } from "react";
import { Download, FolderOpen, Loader2, Trash2 } from "lucide-react";
import { api } from "../api";
import type {
  NodeVersionInfo,
  PHPVersionInfo,
  PythonVersionInfo,
} from "../types";
import { cn } from "../lib/cn";
import { Button } from "./ui/button";
import { Combobox, type ComboboxOption } from "./ui/combobox";

export type VersionInfo = NodeVersionInfo | PHPVersionInfo | PythonVersionInfo;

function formatBytes(n: number): string {
  if (n <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.min(Math.floor(Math.log(n) / Math.log(1024)), units.length - 1);
  return `${(n / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function RuntimeVersionActions<T extends VersionInfo>({
  v,
  busy,
  onInstall,
  onRemove,
}: {
  v: T;
  busy: boolean;
  onInstall: (v: T) => void;
  onRemove: (v: T) => void;
}) {
  return (
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
            <Download className="size-3.5 mr-1.5" />
          )}
          {busy ? "" : "Download"}
        </Button>
      )}
    </div>
  );
}

export function RuntimeVersionRow<T extends VersionInfo>({
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
      <RuntimeVersionActions
        v={v}
        busy={busy}
        onInstall={onInstall}
        onRemove={onRemove}
      />
    </div>
  );
}

export function RuntimeVersionPicker<T extends VersionInfo>({
  label,
  versions,
  busy,
  onInstall,
  onRemove,
}: {
  label: string;
  versions: T[];
  busy: string | null;
  onInstall: (v: T) => void;
  onRemove: (v: T) => void;
}) {
  const [sel, setSel] = useState<string | null>(null);
  const selId = sel ?? (versions.find((v) => v.installed) ?? versions[0]).id;
  const current = versions.find((v) => v.id === selId) ?? versions[0];
  const options: ComboboxOption[] = versions.map((v) => ({
    value: v.id,
    label: `${label} ${v.version}`,
    keywords: v.installed ? "installed" : "",
    node: (
      <span className="flex items-center gap-2 min-w-0">
        <span
          className={cn(
            "size-2 rounded-full shrink-0",
            v.installed
              ? "bg-emerald-400 shadow-[0_0_8px_#34d399]"
              : "bg-white/20",
          )}
        />
        <span className="font-medium">{v.version}</span>
        <span className="text-[var(--orbit-subtle)] truncate">
          {v.installed ? formatBytes(v.diskBytes) : "not downloaded"}
        </span>
      </span>
    ),
  }));

  return (
    <div className="px-4 py-3 flex items-center gap-2">
      <div className="flex-1 min-w-0">
        <Combobox
          options={options}
          value={selId}
          onChange={setSel}
          placeholder="Select version"
          searchPlaceholder="Search versions…"
        />
      </div>
      <RuntimeVersionActions
        v={current}
        busy={busy === current.version}
        onInstall={onInstall}
        onRemove={onRemove}
      />
    </div>
  );
}
