import { useEffect, useMemo, useState } from "react";
import { Play, Table2, TriangleAlert } from "lucide-react";
import { api } from "../api";
import type { QueryResult } from "../types";
import { cn } from "../lib/cn";
import { SqlEditor } from "../components/SqlEditor";

export function DatabaseBrowser() {
  const [databases, setDatabases] = useState<string[]>([]);
  const [database, setDatabase] = useState<string>("");
  const [tables, setTables] = useState<string[]>([]);
  const [schema, setSchema] = useState<Record<string, string[]>>({});
  const [sql, setSql] = useState("");
  const [allowWrites, setAllowWrites] = useState(false);
  const [result, setResult] = useState<QueryResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api
      .dbDatabases()
      .then((dbs) => {
        setDatabases(dbs);
        setDatabase((d) => d || dbs.find((x) => x !== "postgres") || dbs[0] || "");
      })
      .catch((e) => setError(e?.message ?? String(e)));
  }, []);

  useEffect(() => {
    if (!database) return;
    setError(null);
    setResult(null);
    api
      .dbTables(database)
      .then(setTables)
      .catch((e) => {
        setTables([]);
        setError(e?.message ?? String(e));
      });
    api.dbSchema(database).then(setSchema).catch(() => setSchema({}));
  }, [database]);

  const run = async (query?: string) => {
    const q = (query ?? sql).trim();
    if (!q || !database) return;
    setBusy(true);
    setError(null);
    try {
      setResult(await api.dbQuery(database, q, allowWrites));
    } catch (e: any) {
      setResult(null);
      setError(e?.message ?? String(e));
    } finally {
      setBusy(false);
    }
  };

  const selectTable = (t: string) => {
    const q = `SELECT * FROM ${t} LIMIT 100`;
    setSql(q);
    run(q);
  };

  return (
    <div className="h-full flex flex-col p-6 gap-4 overflow-hidden">
      <header className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-lg font-semibold">Database</h1>
          <p className="text-[12px] text-[var(--orbit-muted)]">
            Browse the Postgres database Orbit provisions per project.
          </p>
        </div>
        <select
          value={database}
          onChange={(e) => setDatabase(e.target.value)}
          className="h-8 px-2 rounded-md bg-white/5 border border-white/10 text-[12px] outline-none font-mono"
        >
          {databases.map((db) => (
            <option key={db} value={db}>
              {db}
            </option>
          ))}
        </select>
      </header>

      <div className="flex-1 min-h-0 grid grid-cols-[200px_1fr] gap-4">
        <aside className="min-h-0 overflow-auto scrollbar-thin rounded-xl border border-[var(--orbit-border)] bg-white/2 p-2">
          <p className="px-2 py-1 text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)]">
            Tables
          </p>
          {tables.map((t) => (
            <button
              key={t}
              onClick={() => selectTable(t)}
              className="w-full text-left h-7 px-2 rounded-md flex items-center gap-2 text-[12px] font-mono text-[var(--orbit-text)]/85 hover:bg-white/[0.06] transition"
            >
              <Table2 className="size-3.5 shrink-0 text-[var(--orbit-muted)]" />
              <span className="truncate">{t}</span>
            </button>
          ))}
          {tables.length === 0 && !error && (
            <p className="px-2 text-[11px] text-[var(--orbit-subtle)] italic">No tables.</p>
          )}
        </aside>

        <section className="min-h-0 flex flex-col gap-3">
          <div className="flex items-start gap-2">
            <div className="flex-1 min-w-0">
              <SqlEditor value={sql} onChange={setSql} onRun={() => run()} schema={schema} />
            </div>
            <div className="flex flex-col gap-2">
              <button
                onClick={() => run()}
                disabled={busy}
                className="inline-flex items-center gap-1.5 h-8 px-3 rounded-md bg-[var(--orbit-accent-2)]/15 text-[var(--orbit-accent-2)] border border-[var(--orbit-accent-2)]/30 text-[12px] hover:bg-[var(--orbit-accent-2)]/25 transition disabled:opacity-50"
              >
                <Play className="size-3.5" />
                Run
              </button>
              <label
                className={cn(
                  "inline-flex items-center gap-1.5 h-8 px-2 rounded-md text-[11px] cursor-pointer border",
                  allowWrites
                    ? "border-rose-400/30 bg-rose-400/10 text-rose-300"
                    : "border-white/10 text-[var(--orbit-muted)]",
                )}
              >
                <input
                  type="checkbox"
                  checked={allowWrites}
                  onChange={(e) => setAllowWrites(e.target.checked)}
                  className="accent-rose-400"
                />
                {allowWrites && <TriangleAlert className="size-3" />}
                writes
              </label>
            </div>
          </div>

          {error && (
            <p className="text-[11px] text-rose-300 font-mono whitespace-pre-wrap">{error}</p>
          )}

          <div className="flex-1 min-h-0 overflow-auto scrollbar-thin rounded-lg border border-[var(--orbit-border)]">
            {result && <ResultTable result={result} />}
          </div>
        </section>
      </div>
    </div>
  );
}

function ResultTable({ result }: { result: QueryResult }) {
  const cols = useMemo(() => result.columns, [result]);
  return (
    <table className="w-full text-[12px] font-mono">
      <thead className="sticky top-0 bg-[rgba(28,28,34,0.95)]">
        <tr className="text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)]">
          {cols.map((c) => (
            <th key={c} className="text-left font-semibold px-3 py-2 whitespace-nowrap">
              {c}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {result.rows.map((row, i) => (
          <tr key={i} className="border-t border-white/5 hover:bg-white/[0.03]">
            {row.map((v, j) => (
              <td key={j} className="px-3 py-1.5 whitespace-nowrap max-w-xs truncate">
                {v}
              </td>
            ))}
          </tr>
        ))}
        {result.rows.length === 0 && (
          <tr>
            <td colSpan={cols.length} className="px-3 py-4 text-center text-[var(--orbit-muted)] italic">
              No rows.
            </td>
          </tr>
        )}
      </tbody>
    </table>
  );
}
