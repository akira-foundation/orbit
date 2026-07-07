import { useEffect, useState } from "react";
import { Box, Download, FolderClosed } from "lucide-react";
import { api } from "../api";
import type { S3Object } from "../types";

function formatSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export function StorageBrowser() {
  const [buckets, setBuckets] = useState<string[]>([]);
  const [bucket, setBucket] = useState<string>("");
  const [objects, setObjects] = useState<S3Object[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .bucketList()
      .then((bs) => {
        setBuckets(bs);
        if (bs.length > 0) setBucket((b) => b || bs[0]);
      })
      .catch((e) => setError(e?.message ?? String(e)));
  }, []);

  useEffect(() => {
    if (!bucket) return;
    setError(null);
    api
      .bucketObjects(bucket, "")
      .then(setObjects)
      .catch((e) => {
        setObjects([]);
        setError(e?.message ?? String(e));
      });
  }, [bucket]);

  const download = async (key: string) => {
    try {
      const url = await api.objectURL(bucket, key);
      await api.openURL(url);
    } catch (e: any) {
      setError(e?.message ?? String(e));
    }
  };

  return (
    <div className="h-full flex flex-col p-6 gap-4 overflow-hidden">
      <header>
        <h1 className="text-lg font-semibold">Storage</h1>
        <p className="text-[12px] text-[var(--orbit-muted)]">
          Browse MinIO buckets and objects. One bucket per project.
        </p>
      </header>

      {error && (
        <p className="text-[11px] text-rose-300 font-mono whitespace-pre-wrap">{error}</p>
      )}

      <div className="flex-1 min-h-0 grid grid-cols-[200px_1fr] gap-4">
        <aside className="min-h-0 overflow-auto scrollbar-thin rounded-xl border border-[var(--orbit-border)] bg-white/2 p-2">
          <p className="px-2 py-1 text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)]">
            Buckets
          </p>
          {buckets.map((b) => (
            <button
              key={b}
              onClick={() => setBucket(b)}
              className={
                "w-full text-left h-7 px-2 rounded-md flex items-center gap-2 text-[12px] font-mono transition " +
                (b === bucket ? "bg-white/[0.10] text-white" : "text-[var(--orbit-text)]/85 hover:bg-white/[0.06]")
              }
            >
              <FolderClosed className="size-3.5 shrink-0 text-[var(--orbit-muted)]" />
              <span className="truncate">{b}</span>
            </button>
          ))}
          {buckets.length === 0 && !error && (
            <p className="px-2 text-[11px] text-[var(--orbit-subtle)] italic">No buckets.</p>
          )}
        </aside>

        <section className="min-h-0 overflow-auto scrollbar-thin rounded-lg border border-[var(--orbit-border)]">
          <table className="w-full text-[12px]">
            <thead className="sticky top-0 bg-[rgba(28,28,34,0.95)]">
              <tr className="text-[10px] uppercase tracking-widest text-[var(--orbit-subtle)]">
                <th className="text-left font-semibold px-3 py-2">Key</th>
                <th className="text-right font-semibold px-3 py-2">Size</th>
                <th className="text-left font-semibold px-3 py-2">Modified</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {objects.map((o) => (
                <tr key={o.key} className="border-t border-white/5 hover:bg-white/[0.03]">
                  <td className="px-3 py-1.5 font-mono flex items-center gap-2">
                    <Box className="size-3.5 shrink-0 text-[var(--orbit-muted)]" />
                    <span className="truncate max-w-md">{o.key}</span>
                  </td>
                  <td className="px-3 py-1.5 text-right font-mono text-[var(--orbit-muted)]">
                    {formatSize(o.size)}
                  </td>
                  <td className="px-3 py-1.5 text-[var(--orbit-muted)]">
                    {new Date(o.lastModified).toLocaleString()}
                  </td>
                  <td className="px-3 py-1.5 text-right">
                    <button
                      onClick={() => download(o.key)}
                      title="Download"
                      className="text-[var(--orbit-muted)] hover:text-[var(--orbit-accent-2)] transition-colors"
                    >
                      <Download className="size-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
              {objects.length === 0 && !error && (
                <tr>
                  <td colSpan={4} className="px-3 py-4 text-center text-[var(--orbit-muted)] italic">
                    Empty bucket.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </section>
      </div>
    </div>
  );
}
