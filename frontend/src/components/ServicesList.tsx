import { useEffect, useState } from "react";
import { Play, Square, Trash2, SlidersHorizontal, ExternalLink } from "lucide-react";
import { api } from "../api";
import type { ServiceInfo } from "../types";
import { cn } from "../lib/cn";
import { Button } from "./ui/button";
import { ServiceSetupDialog } from "./ServiceSetupDialog";
import { ConfirmDialog } from "./ConfirmDialog";

const STATUS_DOT: Record<ServiceInfo["status"], string> = {
  running: "bg-emerald-400 shadow-[0_0_8px_#34d399]",
  starting: "bg-amber-400 shadow-[0_0_8px_#fbbf24]",
  error: "bg-rose-400 shadow-[0_0_8px_#fb7185]",
  stopped: "bg-white/20",
};

export function ServicesList() {
  const [items, setItems] = useState<ServiceInfo[]>([]);
  const [busy, setBusy] = useState<string | null>(null);
  const [setupFor, setSetupFor] = useState<ServiceInfo | null>(null);
  const [removeFor, setRemoveFor] = useState<ServiceInfo | null>(null);

  async function refresh() {
    setItems(await api.listServices());
  }

  useEffect(() => {
    refresh();
    const t = setInterval(refresh, 2000);
    return () => clearInterval(t);
  }, []);

  async function toggle(svc: ServiceInfo) {
    setBusy(svc.engine);
    try {
      if (svc.status === "running") await api.stopService(svc.engine);
      else await api.startService(svc.engine);
      await refresh();
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="space-y-2.5">
      {items.map((svc) => (
        <div
          key={svc.engine}
          className="flex items-center justify-between gap-4 rounded-lg border border-white/10 bg-white/[0.03] px-4 py-3"
        >
          <div className="flex items-start gap-3 min-w-0">
            <span
              className={cn(
                "mt-1.5 size-2 rounded-full shrink-0",
                STATUS_DOT[svc.status],
              )}
            />
            <div className="min-w-0 space-y-0.5">
              <p className="text-[13px] font-medium">{svc.displayName}</p>
              {svc.description ? (
                <p className="text-[11px] text-[var(--orbit-muted)] leading-relaxed">
                  {svc.description}
                </p>
              ) : null}
              <p className="text-[11px] text-[var(--orbit-subtle)]">
                {svc.version} ·{" "}
                {svc.installed ? "installed" : "not downloaded"} · {svc.status}
                {svc.refs > 0 ? ` · ${svc.refs} project(s)` : ""}
              </p>
              {svc.webUrl ? (
                <button
                  className="inline-flex items-center gap-1 text-[11px] text-[var(--orbit-accent-2)] hover:underline"
                  onClick={() => api.openURL(`http://${svc.webUrl}`)}
                >
                  {svc.webUrl}
                  <ExternalLink className="size-3" />
                </button>
              ) : null}
            </div>
          </div>

          <div className="flex items-center gap-1.5 shrink-0">
            <Button variant="ghost" size="sm" onClick={() => setSetupFor(svc)}>
              <SlidersHorizontal className="size-3.5 mr-1.5" />
              Setup
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={busy === svc.engine}
              onClick={() => toggle(svc)}
            >
              {svc.status === "running" ? (
                <Square className="size-3.5 mr-1.5" />
              ) : (
                <Play className="size-3.5 mr-1.5" />
              )}
              {svc.status === "running" ? "Stop" : "Start"}
            </Button>
            {svc.installed ? (
              <Button
                variant="ghost"
                size="sm"
                className="text-rose-300 hover:bg-rose-500/10 hover:text-rose-200"
                onClick={() => setRemoveFor(svc)}
                title="Uninstall"
              >
                <Trash2 className="size-3.5" />
              </Button>
            ) : null}
          </div>
        </div>
      ))}

      {setupFor ? (
        <ServiceSetupDialog
          engine={setupFor.engine}
          displayName={setupFor.displayName}
          open={!!setupFor}
          onOpenChange={(o) => !o && setSetupFor(null)}
        />
      ) : null}

      <ConfirmDialog
        open={!!removeFor}
        onOpenChange={(o) => !o && setRemoveFor(null)}
        title={`Uninstall ${removeFor?.displayName ?? ""}?`}
        description="This stops the service and deletes its binary and captured data. You can download it again later."
        confirmLabel="Uninstall"
        destructive
        onConfirm={async () => {
          if (removeFor) {
            await api.uninstallService(removeFor.engine);
            await refresh();
          }
        }}
      />
    </div>
  );
}
