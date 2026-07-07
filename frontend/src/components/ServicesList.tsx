import { useEffect, useState } from "react";
import { api } from "../api";
import type { ServiceInfo } from "../types";
import {
  ServiceRow,
  ServiceVersionCard,
  type RowActions,
} from "./ServiceRows";
import { ServiceSetupDialog } from "./ServiceSetupDialog";
import { ConfirmDialog } from "./ConfirmDialog";

const FAMILY_LABELS: Record<string, string> = { postgres: "PostgreSQL" };

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
