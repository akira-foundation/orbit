import { useEffect, useState } from "react";
import {
  Play,
  Square,
  Trash2,
  SlidersHorizontal,
  ExternalLink,
  AlertTriangle,
  Download,
} from "lucide-react";
import { api } from "../api";
import type { ServiceInfo } from "../types";
import { cn } from "../lib/cn";
import { Button } from "./ui/button";
import { Combobox, type ComboboxOption } from "./ui/combobox";
import { ServiceSetupDialog } from "./ServiceSetupDialog";
import { ConfirmDialog } from "./ConfirmDialog";

const STATUS_DOT: Record<ServiceInfo["status"], string> = {
  running: "bg-emerald-400 shadow-[0_0_8px_#34d399]",
  starting: "bg-amber-400 shadow-[0_0_8px_#fbbf24]",
  error: "bg-rose-400 shadow-[0_0_8px_#fb7185]",
  external: "bg-amber-400 shadow-[0_0_8px_#fbbf24]",
  stopped: "bg-white/20",
};

const FAMILY_LABELS: Record<string, string> = { postgres: "PostgreSQL" };

interface RowActions {
  busy: string | null;
  onToggle: (svc: ServiceInfo) => void;
  onSetup: (svc: ServiceInfo) => void;
  onRemove: (svc: ServiceInfo) => void;
}

function ServiceActions({
  svc,
  actions,
}: {
  svc: ServiceInfo;
  actions: RowActions;
}) {
  return (
    <div className="flex items-center gap-1.5 shrink-0">
      <Button variant="ghost" size="sm" onClick={() => actions.onSetup(svc)}>
        <SlidersHorizontal className="size-3.5 mr-1.5" />
        Setup
      </Button>
      <Button
        variant="outline"
        size="sm"
        disabled={actions.busy === svc.engine || svc.status === "external"}
        title={
          svc.status === "external"
            ? "Port in use by another process, Orbit cannot manage it"
            : undefined
        }
        onClick={() => actions.onToggle(svc)}
      >
        {svc.status === "external" ? (
          <AlertTriangle className="size-3.5 mr-1.5" />
        ) : svc.status === "running" ? (
          <Square className="size-3.5 mr-1.5" />
        ) : svc.installed ? (
          <Play className="size-3.5 mr-1.5" />
        ) : (
          <Download className="size-3.5 mr-1.5" />
        )}
        {svc.status === "external"
          ? "In use"
          : svc.status === "running"
            ? "Stop"
            : svc.installed
              ? "Start"
              : "Download"}
      </Button>
      {svc.installed ? (
        <Button
          variant="ghost"
          size="sm"
          className="text-rose-300 hover:bg-rose-500/10 hover:text-rose-200"
          onClick={() => actions.onRemove(svc)}
          title="Uninstall"
        >
          <Trash2 className="size-3.5" />
        </Button>
      ) : null}
    </div>
  );
}

function ServiceNotices({ svc }: { svc: ServiceInfo }) {
  return (
    <>
      {svc.status === "external" ? (
        <p className="text-[10.5px] text-amber-300/90 leading-relaxed">
          Another process on this machine is already using this port. Orbit
          will not touch it.
        </p>
      ) : null}
      {svc.webUrl ? (
        <button
          className="inline-flex items-center gap-1 text-[11px] text-[var(--orbit-accent-2)] hover:underline"
          onClick={() => api.openURL(`http://${svc.webUrl}`)}
        >
          {svc.webUrl}
          <ExternalLink className="size-3" />
        </button>
      ) : null}
    </>
  );
}

function ServiceRow({
  svc,
  compact,
  actions,
}: {
  svc: ServiceInfo;
  compact: boolean;
  actions: RowActions;
}) {
  return (
    <div
      className={cn(
        "flex items-center justify-between gap-4 px-4 py-3",
        !compact && "rounded-lg border border-white/10 bg-white/[0.03]",
      )}
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
          {!compact && svc.description ? (
            <p className="text-[11px] text-[var(--orbit-muted)] leading-relaxed">
              {svc.description}
            </p>
          ) : null}
          <p className="text-[11px] text-[var(--orbit-subtle)]">
            {svc.version} · {svc.installed ? "installed" : "not downloaded"} ·{" "}
            {svc.status === "external" ? "port in use externally" : svc.status}
            {svc.refs > 0 ? ` · ${svc.refs} project(s)` : ""}
          </p>
          <ServiceNotices svc={svc} />
        </div>
      </div>

      <ServiceActions svc={svc} actions={actions} />
    </div>
  );
}

function ServiceVersionCard({
  familyLabel,
  members,
  actions,
}: {
  familyLabel: string;
  members: ServiceInfo[];
  actions: RowActions;
}) {
  const [sel, setSel] = useState<string | null>(null);
  const pickDefault = () => {
    const active = members.find(
      (m) => m.status === "running" || m.status === "starting",
    );
    if (active) return active.engine;
    const installed = members.find((m) => m.installed);
    return (installed ?? members[0]).engine;
  };
  const selEngine = sel ?? pickDefault();
  const current = members.find((m) => m.engine === selEngine) ?? members[0];
  const options: ComboboxOption[] = members.map((m) => ({
    value: m.engine,
    label: m.version,
    keywords: m.status,
    node: (
      <span className="flex items-center gap-2 min-w-0">
        <span
          className={cn("size-2 rounded-full shrink-0", STATUS_DOT[m.status])}
        />
        <span className="font-medium">{m.version}</span>
        <span className="text-[var(--orbit-subtle)] truncate">
          {m.installed ? "installed" : "not downloaded"} · {m.status}
        </span>
      </span>
    ),
  }));

  return (
    <div className="rounded-lg border border-white/10 bg-white/[0.03]">
      <div className="px-4 pt-3 pb-2 border-b border-white/[0.06] space-y-0.5">
        <p className="text-[13px] font-medium">{familyLabel}</p>
        {members[0].description ? (
          <p className="text-[11px] text-[var(--orbit-muted)] leading-relaxed">
            {members[0].description}
          </p>
        ) : null}
      </div>
      <div className="px-4 py-3 space-y-2">
        <div className="flex items-center gap-2">
          <div className="flex-1 min-w-0">
            <Combobox
              options={options}
              value={selEngine}
              onChange={setSel}
              placeholder="Select version"
              searchPlaceholder="Search versions…"
            />
          </div>
          <ServiceActions svc={current} actions={actions} />
        </div>
        <ServiceNotices svc={current} />
        {current.refs > 0 ? (
          <p className="text-[11px] text-[var(--orbit-subtle)]">
            {current.refs} project(s) using this
          </p>
        ) : null}
      </div>
    </div>
  );
}

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

  const actions: RowActions = {
    busy,
    onToggle: toggle,
    onSetup: setSetupFor,
    onRemove: setRemoveFor,
  };

  const groups = new Map<string, ServiceInfo[]>();
  for (const svc of items) {
    const key = svc.family || svc.engine;
    const list = groups.get(key) ?? [];
    list.push(svc);
    groups.set(key, list);
  }

  return (
    <div className="space-y-2.5">
      {[...groups.entries()].map(([key, members]) =>
        members.length === 1 ? (
          <ServiceRow
            key={key}
            svc={members[0]}
            compact={false}
            actions={actions}
          />
        ) : (
          <ServiceVersionCard
            key={key}
            familyLabel={FAMILY_LABELS[key] ?? key}
            members={members}
            actions={actions}
          />
        ),
      )}

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
