import { useEffect, useState } from "react";
import { ShieldCheck, AlertTriangle, RotateCcw } from "lucide-react";
import { api } from "../api";
import type { SystemStatus } from "../types";
import { cn } from "../lib/cn";

export function SetupBanner() {
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = async () => {
    try {
      setStatus(await api.systemStatus());
    } catch (e: any) {
      setError(e?.message ?? String(e));
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  if (!status || status.os !== "darwin" || status.setup) return null;

  const run = async () => {
    setBusy(true);
    setError(null);
    try {
      await api.systemSetup();
      await refresh();
    } catch (e: any) {
      setError(e?.message ?? String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      className={cn(
        "no-drag mx-2 mb-1.5 mt-1 rounded-xl border px-4 py-3",
        "border-amber-400/30 bg-amber-400/8 text-amber-100",
        "flex items-start gap-3",
      )}
    >
      {status.herdConflict && (
        <div className="size-8 shrink-0 rounded-full bg-amber-400/15 border border-amber-400/30 flex items-center justify-center">
          <AlertTriangle className="size-4 text-amber-300" />
        </div>
      )}
      <div className="min-w-0 flex-1 space-y-1">
        <p className="text-sm font-semibold">
          {status.herdConflict
            ? "Local domain port is held by Herd"
            : "Set up Orbit local domains"}
        </p>
        <p className="text-xs text-amber-200/80">
          {status.herdConflict ? (
            <>
              Open Herd Settings, enable{" "}
              <span className="font-mono">Bind to localhost only</span>, then
              click Retry. Herd's <code>*.test</code> sites keep working.
            </>
          ) : (
            <>
              One-time admin prompt configures dnsmasq, a loopback alias and a
              forwarder so <code className="font-mono">*.orbit.test</code>{" "}
              resolves to Orbit without ports or <code>/etc/hosts</code>.
            </>
          )}
        </p>
        <Checks status={status} />
        {error && (
          <p className="text-xs text-rose-300 mt-1 font-mono">{error}</p>
        )}
      </div>
      <div className="shrink-0 flex items-center gap-2">
        <button
          onClick={refresh}
          disabled={busy}
          className="size-8 inline-flex items-center justify-center rounded-md border border-amber-400/30 hover:bg-amber-400/15 text-amber-200"
          title="Re-check"
        >
          <RotateCcw className="size-3.5" />
        </button>
        <button
          onClick={run}
          disabled={busy}
          className={cn(
            "h-8 px-3 inline-flex items-center gap-1.5 rounded-md text-xs font-semibold",
            "bg-amber-400/85 text-amber-950 hover:bg-amber-300 transition-colors",
            "disabled:opacity-50",
          )}
        >
          <ShieldCheck className="size-3.5" />
          {busy ? "Setting up…" : "Set up"}
        </button>
      </div>
    </div>
  );
}

function Checks({ status }: { status: SystemStatus }) {
  const items: [string, boolean][] = [
    ["Loopback 127.0.0.2", status.loopbackOk],
    ["DNS *.orbit.test", status.dnsmasqOk],
    ["resolver", status.resolverOk],
    ["proxy daemon", status.daemonOk],
  ];
  return (
    <div className="flex flex-wrap gap-1.5 mt-1">
      {items.map(([label, ok]) => (
        <span
          key={label}
          className={cn(
            "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium",
            ok
              ? "bg-emerald-400/10 text-emerald-300 border border-emerald-400/20"
              : "bg-amber-400/10 text-amber-200 border border-amber-400/25",
          )}
        >
          <span
            className={cn(
              "size-1.5 rounded-full",
              ok ? "bg-emerald-400" : "bg-amber-400",
            )}
          />
          {label}
        </span>
      ))}
    </div>
  );
}
