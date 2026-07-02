import { useEffect, useState } from "react";
import { api } from "../api";
import type { ServiceInfo } from "../types";
import { Switch } from "./ui/switch";

const PG_VERSIONS = ["18", "17", "16", "15"];

function ServiceRow({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-lg border border-white/10 bg-white/[0.03] px-4 py-3">
      <div className="min-w-0 space-y-0.5">
        <p className="text-[13px] font-medium">{title}</p>
        {description ? (
          <p className="text-[11px] text-[var(--orbit-muted)] leading-relaxed">
            {description}
          </p>
        ) : null}
      </div>
      <div className="flex items-center gap-2 shrink-0">{children}</div>
    </div>
  );
}

export function ServicesPanel({
  projectId,
  isRunning,
}: {
  projectId: string;
  isRunning: boolean;
}) {
  const [all, setAll] = useState<ServiceInfo[]>([]);
  const [enabled, setEnabled] = useState<string[]>([]);

  async function refresh() {
    const [services, projEnabled] = await Promise.all([
      api.listServices(),
      api.projectServices(projectId),
    ]);
    setAll(services);
    setEnabled(projEnabled);
  }

  useEffect(() => {
    refresh();
  }, [projectId]);

  async function toggle(engine: string, on: boolean) {
    if (on) {
      await api.enableServiceForProject(projectId, engine);
    } else {
      await api.disableServiceForProject(projectId, engine);
    }
    await refresh();
  }

  const simple = all.filter((svc) => svc.family !== "postgres");
  const pgMembers = all.filter((svc) => svc.family === "postgres");
  const pgEnabled = enabled.find((e) => e.startsWith("postgres-")) ?? null;

  async function setPgVersion(major: string) {
    if (pgEnabled) await api.disableServiceForProject(projectId, pgEnabled);
    await api.enableServiceForProject(projectId, `postgres-${major}`);
    await refresh();
  }

  return (
    <div className="flex flex-col gap-2">
      {simple.map((svc) => (
        <ServiceRow
          key={svc.engine}
          title={svc.displayName}
          description={svc.description}
        >
          <Switch
            checked={enabled.includes(svc.engine)}
            disabled={isRunning}
            onCheckedChange={(v) => toggle(svc.engine, v)}
          />
        </ServiceRow>
      ))}

      {pgMembers.length > 0 ? (
        <ServiceRow
          title="PostgreSQL"
          description={pgMembers[0].description}
        >
          <select
            disabled={!pgEnabled || isRunning}
            value={pgEnabled?.replace("postgres-", "") ?? "18"}
            onChange={(e) => setPgVersion(e.target.value)}
            className="h-7 rounded-md border border-white/10 bg-white/[0.04] px-1.5 text-[11px] disabled:opacity-50"
          >
            {PG_VERSIONS.map((v) => (
              <option key={v} value={v} className="bg-zinc-900">
                {v}
              </option>
            ))}
          </select>
          <Switch
            checked={!!pgEnabled}
            disabled={isRunning}
            onCheckedChange={(v) =>
              v
                ? toggle("postgres-18", true)
                : toggle(pgEnabled ?? "postgres-18", false)
            }
          />
        </ServiceRow>
      ) : null}

      {isRunning ? (
        <p className="text-[10px] text-[var(--orbit-subtle)]">
          Stop the project to change its services.
        </p>
      ) : null}
    </div>
  );
}
