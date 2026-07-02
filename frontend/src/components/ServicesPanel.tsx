import { useEffect, useState } from "react";
import { api } from "../api";
import type { ServiceInfo } from "../types";

const PG_VERSIONS = ["18", "17", "16", "15"];

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
  const hasPostgres = all.some((svc) => svc.family === "postgres");
  const pgEnabled = enabled.find((e) => e.startsWith("postgres-")) ?? null;

  async function setPgVersion(major: string) {
    if (pgEnabled) await api.disableServiceForProject(projectId, pgEnabled);
    await api.enableServiceForProject(projectId, `postgres-${major}`);
    await refresh();
  }

  return (
    <div className="flex flex-col gap-2">
      {simple.map((svc) => {
        const on = enabled.includes(svc.engine);
        return (
          <label
            key={svc.engine}
            className="flex items-center justify-between rounded-lg border border-white/10 px-3 py-2"
          >
            <span className="text-sm">{svc.displayName}</span>
            <input
              type="checkbox"
              checked={on}
              onChange={(e) => toggle(svc.engine, e.target.checked)}
            />
          </label>
        );
      })}

      {hasPostgres ? (
        <label className="flex items-center justify-between rounded-lg border border-white/10 px-3 py-2">
          <span className="text-sm">PostgreSQL</span>
          <span className="flex items-center gap-2">
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
            <input
              type="checkbox"
              checked={!!pgEnabled}
              disabled={isRunning}
              onChange={(e) =>
                e.target.checked
                  ? toggle("postgres-18", true)
                  : toggle(pgEnabled ?? "postgres-18", false)
              }
            />
          </span>
        </label>
      ) : null}

      {isRunning ? (
        <p className="text-[10px] text-[var(--orbit-subtle)]">
          Stop the project to change its database version.
        </p>
      ) : null}
    </div>
  );
}
