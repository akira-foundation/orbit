import { useEffect, useState } from "react";
import { api } from "../api";
import type { ServiceInfo } from "../types";

export function ServicesPanel({ projectId }: { projectId: string }) {
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

  return (
    <div className="flex flex-col gap-2">
      <h2 className="text-sm font-semibold text-zinc-300">Services</h2>
      {all.map((svc) => {
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
    </div>
  );
}
