import { useEffect } from "react";
import { EventsOn } from "../../wailsjs/runtime/runtime";

export function useWailsEvent<T = unknown>(
  events: string | string[],
  handler: (data: T) => void,
  enabled = true,
) {
  useEffect(() => {
    if (!enabled) return;
    const names = Array.isArray(events) ? events : [events];
    const offs = names.map((n) => EventsOn(n, handler));
    return () => offs.forEach((off) => off());
  }, [Array.isArray(events) ? events.join("|") : events, handler, enabled]);
}
