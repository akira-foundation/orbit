import { useEffect, useRef } from "react";
import type { Terminal } from "@xterm/xterm";
import { api } from "../api";
import { useWailsEvent } from "./useWailsEvent";

export function useProjectTerminal(projectId: string, term: Terminal | null) {
  const replayedRef = useRef(false);
  const queueRef = useRef<string[]>([]);
  const termRef = useRef<Terminal | null>(null);
  termRef.current = term;

  useWailsEvent<string>(`terminal:data:${projectId}`, (chunk) => {
    if (!replayedRef.current) {
      queueRef.current.push(chunk);
      return;
    }
    termRef.current?.write(chunk);
  });

  useEffect(() => {
    if (!term) return;
    replayedRef.current = false;
    queueRef.current = [];

    let cancelled = false;

    async function boot() {
      await api.terminalStart(projectId);
      const buffer = await api.terminalBuffer(projectId);
      if (cancelled) return;
      if (buffer) term!.write(buffer);
      for (const chunk of queueRef.current) {
        term!.write(chunk);
      }
      queueRef.current = [];
      replayedRef.current = true;
    }

    boot();

    const onData = term.onData((data) => {
      api.terminalWrite(projectId, data);
    });

    return () => {
      cancelled = true;
      onData.dispose();
    };
  }, [projectId, term]);
}
