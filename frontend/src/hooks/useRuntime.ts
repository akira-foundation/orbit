import { useCallback, useEffect, useRef, useState } from "react";
import { useInterval } from "usehooks-ts";
import { api } from "../api";
import { useWailsEvent } from "./useWailsEvent";
import type {
  RuntimeLogEvent,
  RuntimeLogLine,
  RuntimeSnapshot,
  RuntimeStatusEvent,
} from "../types";

const MAX_LOGS = 2000;
const STATUS_EVENTS = [
  "runtime:starting",
  "runtime:running",
  "runtime:stopped",
  "runtime:error",
];

interface UseRuntime {
  snapshot: RuntimeSnapshot | null;
  logs: RuntimeLogLine[];
  uptimeMs: number;
  start: () => Promise<void>;
  stop: () => Promise<void>;
  restart: () => Promise<void>;
  clearLogs: () => void;
}

export function useRuntime(projectId: string | null): UseRuntime {
  const [snapshot, setSnapshot] = useState<RuntimeSnapshot | null>(null);
  const [logs, setLogs] = useState<RuntimeLogLine[]>([]);
  const [uptimeMs, setUptimeMs] = useState(0);

  const startedAtRef = useRef<number | null>(null);
  const statusRef = useRef<string>("stopped");

  useEffect(() => {
    if (!projectId) {
      setSnapshot(null);
      setLogs([]);
      setUptimeMs(0);
      return;
    }
    let cancelled = false;
    Promise.all([api.runtimeStatus(projectId), api.runtimeLogs(projectId)])
      .then(([snap, ls]) => {
        if (cancelled) return;
        setSnapshot(snap);
        setLogs(ls);
        startedAtRef.current = snap.startedAt
          ? new Date(snap.startedAt).getTime()
          : null;
        statusRef.current = snap.status;
      })
      .catch(() => {
        if (cancelled) return;
        setSnapshot(null);
        setLogs([]);
      });
    return () => {
      cancelled = true;
    };
  }, [projectId]);

  const onStatus = useCallback(
    (e: RuntimeStatusEvent) => {
      if (!projectId || e.projectId !== projectId) return;
      setSnapshot(e.snapshot);
      startedAtRef.current = e.snapshot.startedAt
        ? new Date(e.snapshot.startedAt).getTime()
        : null;
      statusRef.current = e.snapshot.status;
    },
    [projectId],
  );

  const onLog = useCallback(
    (e: RuntimeLogEvent) => {
      if (!projectId || e.projectId !== projectId) return;
      setLogs((prev) => {
        const next =
          prev.length >= MAX_LOGS ? prev.slice(prev.length - MAX_LOGS + 1) : prev.slice();
        next.push(e.line);
        return next;
      });
    },
    [projectId],
  );

  useWailsEvent<RuntimeStatusEvent>(STATUS_EVENTS, onStatus, !!projectId);
  useWailsEvent<RuntimeLogEvent>("runtime:log", onLog, !!projectId);

  useInterval(() => {
    if (
      startedAtRef.current &&
      (statusRef.current === "running" || statusRef.current === "starting")
    ) {
      setUptimeMs(Date.now() - startedAtRef.current);
    } else if (uptimeMs !== 0) {
      setUptimeMs(0);
    }
  }, 1000);

  return {
    snapshot,
    logs,
    uptimeMs,
    start: async () => {
      if (!projectId) return;
      await api.startProject(projectId);
    },
    stop: async () => {
      if (!projectId) return;
      await api.stopProject(projectId);
    },
    restart: async () => {
      if (!projectId) return;
      await api.restartProject(projectId);
    },
    clearLogs: () => setLogs([]),
  };
}

export function formatUptime(ms: number): string {
  if (ms <= 0) return "—";
  const s = Math.floor(ms / 1000);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  if (h > 0) return `${h}h ${m}m ${sec}s`;
  if (m > 0) return `${m}m ${sec}s`;
  return `${sec}s`;
}
