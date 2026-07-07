import { useEffect, useState } from "react";
import { Eye, EyeOff, Plus, Save, Search, Trash2 } from "lucide-react";
import { api } from "../api";
import { Button } from "./ui/button";
import { cn } from "../lib/cn";

interface Row {
  key: string;
  value: string;
}

const SECRET_RE = /(SECRET|KEY|PASSWORD|TOKEN)/i;

function toRows(vars: Record<string, string>): Row[] {
  return Object.entries(vars)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, value]) => ({ key, value }));
}

export function EnvEditorPanel({ projectId }: { projectId: string }) {
  const [rows, setRows] = useState<Row[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [revealed, setRevealed] = useState<Set<number>>(new Set());
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState("");

  useEffect(() => {
    let alive = true;
    api
      .projectEnv(projectId)
      .then((vars) => {
        if (alive) {
          setRows(toRows(vars));
          setLoaded(true);
        }
      })
      .catch((e) => {
        if (alive) {
          setError(e?.message ?? String(e));
          setLoaded(true);
        }
      });
    return () => {
      alive = false;
    };
  }, [projectId]);

  const update = (i: number, patch: Partial<Row>) => {
    setSaved(false);
    setRows((rs) => rs.map((r, j) => (j === i ? { ...r, ...patch } : r)));
  };

  const removeRow = (i: number) => {
    setSaved(false);
    setRows((rs) => rs.filter((_, j) => j !== i));
  };

  const addRow = () => {
    setSaved(false);
    setRows((rs) => [...rs, { key: "", value: "" }]);
  };

  const toggleReveal = (i: number) =>
    setRevealed((set) => {
      const next = new Set(set);
      if (next.has(i)) next.delete(i);
      else next.add(i);
      return next;
    });

  const save = async () => {
    setSaving(true);
    setError(null);
    try {
      const vars: Record<string, string> = {};
      for (const r of rows) {
        const key = r.key.trim();
        if (key) vars[key] = r.value;
      }
      await api.saveProjectEnv(projectId, vars);
      setSaved(true);
    } catch (e: any) {
      setError(e?.message ?? String(e));
    } finally {
      setSaving(false);
    }
  };

  if (!loaded) {
    return (
      <p className="text-[12px] text-[var(--orbit-muted)]">Loading .env…</p>
    );
  }

  const q = query.trim().toLowerCase();
  const visible = rows
    .map((r, i) => ({ r, i }))
    .filter(
      ({ r }) =>
        !q ||
        r.key.toLowerCase().includes(q) ||
        r.value.toLowerCase().includes(q),
    );

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-[11px] text-[var(--orbit-muted)]">
          Edits write to the project&apos;s <code className="font-mono">.env</code>. Restart the
          project to apply.
        </p>
        <Button size="sm" onClick={save} disabled={saving}>
          <Save className="size-3.5" />
          {saving ? "Saving…" : saved ? "Saved" : "Save"}
        </Button>
      </div>

      {error && (
        <p className="text-[11px] text-rose-300">{error}</p>
      )}

      {rows.length > 0 && (
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-[var(--orbit-subtle)]" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter variables…"
            className="h-8 w-full pl-8 pr-2.5 rounded-md bg-white/5 border border-white/10 text-[12px] outline-none focus:border-[var(--orbit-accent-2)]/50 placeholder:text-[var(--orbit-subtle)]"
          />
        </div>
      )}

      <div className="space-y-1.5">
        {visible.map(({ r, i }) => {
          const secret = SECRET_RE.test(r.key);
          const hidden = secret && !revealed.has(i);
          return (
            <div key={i} className="flex items-center gap-1.5">
              <input
                value={r.key}
                onChange={(e) => update(i, { key: e.target.value })}
                placeholder="KEY"
                className="h-8 w-1/3 px-2.5 rounded-md bg-white/5 border border-white/10 text-[12px] font-mono outline-none focus:border-[var(--orbit-accent-2)]/50 placeholder:text-[var(--orbit-subtle)]"
              />
              <div className="relative flex-1">
                <input
                  value={r.value}
                  onChange={(e) => update(i, { value: e.target.value })}
                  type={hidden ? "password" : "text"}
                  placeholder="value"
                  className={cn(
                    "h-8 w-full px-2.5 rounded-md bg-white/5 border border-white/10 text-[12px] font-mono outline-none focus:border-[var(--orbit-accent-2)]/50 placeholder:text-[var(--orbit-subtle)]",
                    secret && "pr-9",
                  )}
                />
                {secret && (
                  <button
                    onClick={() => toggleReveal(i)}
                    title={hidden ? "Reveal" : "Hide"}
                    className="absolute right-1 top-1/2 -translate-y-1/2 size-6 inline-flex items-center justify-center rounded text-[var(--orbit-muted)] hover:text-[var(--orbit-text)] hover:bg-white/5 transition-colors"
                  >
                    {hidden ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                  </button>
                )}
              </div>
              <button
                onClick={() => removeRow(i)}
                title="Remove"
                className={cn(
                  "shrink-0 size-8 inline-flex items-center justify-center rounded-md transition-colors",
                  "text-[var(--orbit-muted)] hover:text-rose-300 hover:bg-white/5",
                )}
              >
                <Trash2 className="size-3.5" />
              </button>
            </div>
          );
        })}
        {rows.length === 0 && (
          <p className="text-[12px] text-[var(--orbit-muted)] italic">
            No variables. Add one below.
          </p>
        )}
        {rows.length > 0 && visible.length === 0 && (
          <p className="text-[12px] text-[var(--orbit-muted)] italic">
            No variables match &ldquo;{query}&rdquo;.
          </p>
        )}
      </div>

      <button
        onClick={addRow}
        className="inline-flex items-center gap-1.5 h-8 px-3 rounded-md text-[12px] text-[var(--orbit-muted)] hover:text-[var(--orbit-text)] border border-dashed border-white/10 hover:border-white/20 transition-colors"
      >
        <Plus className="size-3.5" />
        Add variable
      </button>
    </div>
  );
}
