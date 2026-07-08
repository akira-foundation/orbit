import { useEffect, useRef } from "react";
import { api } from "../api";

export function usePrewarm(enabled: boolean, hoverMs: number) {
  const lastWarm = useRef<Map<string, number>>(new Map());
  const hoverTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (hoverTimer.current) clearTimeout(hoverTimer.current);
    };
  }, []);

  const prewarm = (id: string) => {
    if (!enabled) return;
    const now = Date.now();
    if (now - (lastWarm.current.get(id) ?? 0) < 15000) return;
    lastWarm.current.set(id, now);
    api.prewarm(id).catch(() => {});
  };

  const onRowEnter = (id: string) => {
    if (hoverTimer.current) clearTimeout(hoverTimer.current);
    hoverTimer.current = setTimeout(() => prewarm(id), hoverMs);
  };

  const onRowLeave = () => {
    if (hoverTimer.current) {
      clearTimeout(hoverTimer.current);
      hoverTimer.current = null;
    }
  };

  return { prewarm, onRowEnter, onRowLeave };
}
