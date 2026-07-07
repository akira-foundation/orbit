import { useEffect, useState } from "react";
import { Check, Copy } from "lucide-react";
import { api } from "../api";
import type { ShareInfo } from "../types";
import { Switch } from "./ui/switch";

export function SharePanel({ projectId }: { projectId: string }) {
  const [info, setInfo] = useState<ShareInfo | null>(null);
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let alive = true;
    api
      .shareInfo(projectId)
      .then((i) => alive && setInfo(i))
      .catch((e) => alive && setError(e?.message ?? String(e)));
    return () => {
      alive = false;
    };
  }, [projectId]);

  const toggle = async (next: boolean) => {
    setBusy(true);
    setError(null);
    try {
      if (next) {
        setInfo(await api.enableLANShare(projectId));
      } else {
        await api.disableLANShare(projectId);
        setInfo({ enabled: false, url: "", qr: "" });
      }
    } catch (e: any) {
      setError(e?.message ?? String(e));
    } finally {
      setBusy(false);
    }
  };

  const copy = async () => {
    if (!info?.url) return;
    await navigator.clipboard.writeText(info.url);
    setCopied(true);
    setTimeout(() => setCopied(false), 1200);
  };

  const enabled = !!info?.enabled;

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-4 rounded-lg border border-white/10 bg-white/[0.03] p-4">
        <div className="min-w-0 space-y-1">
          <p className="text-[13px] font-medium">LAN sharing</p>
          <p className="text-[11px] text-[var(--orbit-muted)] leading-relaxed">
            Serve this project to other devices on your Wi-Fi. Anyone on the network can
            open it while this is on.
          </p>
        </div>
        <Switch
          checked={enabled}
          disabled={busy}
          onCheckedChange={toggle}
          className="mt-0.5 shrink-0"
        />
      </div>

      {error && (
        <p className="text-[11px] text-rose-300 font-mono whitespace-pre-wrap">{error}</p>
      )}

      {enabled && info?.url && (
        <div className="space-y-3">
          <div className="flex items-center gap-2 rounded-lg border border-white/10 bg-white/[0.03] px-3 py-2">
            <span className="flex-1 min-w-0 truncate font-mono text-[12px] text-[var(--orbit-text)]">
              {info.url}
            </span>
            <button
              onClick={copy}
              title="Copy link"
              className="shrink-0 text-[var(--orbit-muted)] hover:text-[var(--orbit-text)] transition-colors"
            >
              {copied ? (
                <Check className="size-3.5 text-emerald-300" />
              ) : (
                <Copy className="size-3.5" />
              )}
            </button>
          </div>
          {info.qr ? (
            <div className="flex justify-center">
              <img
                src={info.qr}
                alt="Share QR code"
                className="size-44 rounded-lg bg-white p-2"
              />
            </div>
          ) : (
            <p className="text-center text-[11px] text-[var(--orbit-subtle)]">
              Scan the link on your phone to open it.
            </p>
          )}
        </div>
      )}
    </div>
  );
}
