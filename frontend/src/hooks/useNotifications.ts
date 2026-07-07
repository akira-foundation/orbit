import { useCallback } from "react";
import { useProjects } from "../store";
import { useNotificationStore } from "../stores/notifications";
import { useWailsEvent } from "./useWailsEvent";
import { api } from "../api";
import type { RuntimeStatusEvent } from "../types";

interface TlsExpiringEvent {
  expiresAt: string;
}

export function useNotificationEvents() {
  const projects = useProjects((s) => s.projects);
  const push = useNotificationStore((s) => s.push);
  const enabled = useNotificationStore((s) => s.enabled);

  const emit = useCallback(
    (kind: "crash" | "cert", title: string, body: string) => {
      if (!enabled) return;
      push({ kind, title, body });
      api.notify(title, body).catch(() => {});
    },
    [push, enabled],
  );

  const onCrash = useCallback(
    (e: RuntimeStatusEvent) => {
      const name = projects.find((p) => p.id === e.projectId)?.name ?? "A project";
      const reason = e.snapshot?.error?.trim();
      emit("crash", `${name} stopped`, reason || "The runtime exited unexpectedly.");
    },
    [projects, emit],
  );
  useWailsEvent<RuntimeStatusEvent>("runtime:crash", onCrash);

  const onTlsExpiring = useCallback(
    (e: TlsExpiringEvent) => {
      const when = new Date(e.expiresAt).toLocaleDateString();
      emit("cert", "TLS certificate expiring", `The local HTTPS certificate expires on ${when}.`);
    },
    [emit],
  );
  useWailsEvent<TlsExpiringEvent>("tls:expiring", onTlsExpiring);
}
