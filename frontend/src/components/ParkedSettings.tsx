import { useEffect, useState } from "react";
import { FolderPlus, FolderSearch, X } from "lucide-react";
import { api } from "../api";
import { parkedApi } from "../parkedApi";
import { Button } from "./ui/button";

export function ParkedSettings() {
  const [folders, setFolders] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = () => parkedApi.folders().then(setFolders);

  useEffect(() => {
    refresh();
  }, []);

  const add = async () => {
    setBusy(true);
    setError(null);
    try {
      const path = await api.selectFolder();
      if (path) {
        await parkedApi.addFolder(path);
        await refresh();
      }
    } catch (e: any) {
      setError(e?.message ?? String(e));
    } finally {
      setBusy(false);
    }
  };

  const remove = async (path: string) => {
    await parkedApi.removeFolder(path);
    await refresh();
  };

  return (
    <div className="rounded-lg border border-white/10 bg-white/[0.03] p-4 space-y-3">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 space-y-1">
          <p className="text-[13px] font-medium">Parked folders</p>
          <p className="text-[11px] text-[var(--orbit-muted)] leading-relaxed">
            Orbit watches these folders and registers every project found
            inside them automatically. New projects appear on their own;
            deleted ones are removed.
          </p>
        </div>
        <Button variant="outline" size="sm" disabled={busy} onClick={add}>
          <FolderPlus className="size-3.5 mr-1.5" />
          Add folder
        </Button>
      </div>
      {folders.length > 0 ? (
        <ul className="divide-y divide-white/[0.04] rounded-md border border-white/[0.06]">
          {folders.map((f) => (
            <li
              key={f}
              className="group flex items-center gap-2 px-3 py-2 text-[12px]"
            >
              <FolderSearch className="size-3.5 shrink-0 text-[var(--orbit-muted)]" />
              <span className="font-mono truncate flex-1">{f}</span>
              <button
                onClick={() => remove(f)}
                title="Stop watching (already-added projects stay)"
                className="opacity-0 group-hover:opacity-100 shrink-0 text-[var(--orbit-muted)] hover:text-rose-300 transition"
              >
                <X className="size-3.5" />
              </button>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-[11px] text-[var(--orbit-subtle)]">
          No parked folders yet.
        </p>
      )}
      {error ? (
        <p className="text-[11px] text-rose-300 font-mono whitespace-pre-wrap">
          {error}
        </p>
      ) : null}
    </div>
  );
}
