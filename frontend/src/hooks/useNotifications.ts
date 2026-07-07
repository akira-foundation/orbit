import { useCallback, useEffect } from "react";
import { useProjects } from "../store";
import { useWailsEvent } from "./useWailsEvent";
import type { RuntimeStatusEvent } from "../types";

interface TlsExpiringEvent {
  expiresAt: string;
}

function notify(title: string, body: string) {
  if (typeof Notification === "undefined") return;
  if (Notification.permission === "granted") {
    new Notification(title, { body });
    return;
  }
  if (Notification.permission !== "denied") {
    Notification.requestPermission().then((p) => {
      if (p === "granted") new Notification(title, { body });
    });
  }
}

export function useNotifications() {
  const projects = useProjects((s) => s.projects);

  useEffect(() => {
    if (typeof Notification !== "undefined" && Notification.permission === "default") {
      Notification.requestPermission();
    }
  }, []);

  const onCrash = useCallback(
    (e: RuntimeStatusEvent) => {
      const name = projects.find((p) => p.id === e.projectId)?.name ?? "A project";
      notify("Project crashed", `${name} stopped after repeated restart failures.`);
    },
    [projects],
  );
  useWailsEvent<RuntimeStatusEvent>("runtime:crash", onCrash);

  const onTlsExpiring = useCallback((e: TlsExpiringEvent) => {
    const when = new Date(e.expiresAt).toLocaleDateString();
    notify("Orbit TLS certificate expiring", `The local HTTPS certificate expires on ${when}.`);
  }, []);
  useWailsEvent<TlsExpiringEvent>("tls:expiring", onTlsExpiring);
}
