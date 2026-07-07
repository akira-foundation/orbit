import { useEffect, useRef, useState } from "react";
import { AlertTriangle, Bell, ShieldAlert, X } from "lucide-react";
import { useNotificationStore } from "../stores/notifications";
import type { AppNotification, NotificationKind } from "../types";
import { cn } from "../lib/cn";

const TOAST_MS = 6000;

function iconFor(kind: NotificationKind) {
  if (kind === "crash") return <AlertTriangle className="size-4 text-rose-300" />;
  if (kind === "cert") return <ShieldAlert className="size-4 text-amber-300" />;
  return <Bell className="size-4 text-[var(--orbit-accent-2)]" />;
}

export function Toaster() {
  const items = useNotificationStore((s) => s.items);
  const [active, setActive] = useState<AppNotification[]>([]);
  const seen = useRef<Set<string>>(new Set());

  useEffect(() => {
    const fresh = items.filter((i) => !seen.current.has(i.id));
    if (fresh.length === 0) return;
    fresh.forEach((i) => seen.current.add(i.id));
    setActive((a) => [...fresh, ...a].slice(0, 4));
    const timers = fresh.map((i) =>
      setTimeout(() => setActive((a) => a.filter((x) => x.id !== i.id)), TOAST_MS),
    );
    return () => timers.forEach(clearTimeout);
  }, [items]);

  if (active.length === 0) return null;

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 w-80">
      {active.map((n) => (
        <div
          key={n.id}
          className={cn(
            "rounded-2xl px-3.5 py-3 border bg-[rgba(28,28,34,0.85)] backdrop-blur-2xl backdrop-saturate-150 shadow-[0_24px_64px_rgba(0,0,0,0.6),inset_0_1px_0_rgba(255,255,255,0.08)]",
            n.kind === "crash" ? "border-rose-400/30" : "border-white/10",
          )}
        >
          <div className="flex items-start gap-2.5">
            <span className="mt-0.5 shrink-0">{iconFor(n.kind)}</span>
            <div className="min-w-0 flex-1">
              <p className="text-[13px] font-semibold text-[var(--orbit-text)]">{n.title}</p>
              <p className="mt-0.5 text-[12px] text-[var(--orbit-muted)] leading-snug">{n.body}</p>
            </div>
            <button
              onClick={() => setActive((a) => a.filter((x) => x.id !== n.id))}
              className="shrink-0 text-[var(--orbit-muted)] hover:text-[var(--orbit-text)]"
            >
              <X className="size-3.5" />
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}
