import { useEffect, useRef, useState } from "react";
import { AlertTriangle, Bell, Check, ShieldAlert, Trash2 } from "lucide-react";
import { useNotificationStore } from "../stores/notifications";
import type { NotificationKind } from "../types";
import { cn } from "../lib/cn";

function iconFor(kind: NotificationKind) {
  if (kind === "crash") return <AlertTriangle className="size-3.5 text-rose-300" />;
  if (kind === "cert") return <ShieldAlert className="size-3.5 text-amber-300" />;
  return <Bell className="size-3.5 text-[var(--orbit-accent-2)]" />;
}

function ago(ts: number) {
  const s = Math.floor((Date.now() - ts) / 1000);
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return `${Math.floor(s / 86400)}d ago`;
}

export function NotificationBell() {
  const { items, markAllRead, clear } = useNotificationStore();
  const unread = items.filter((i) => !i.read).length;
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    window.addEventListener("mousedown", onDown);
    return () => window.removeEventListener("mousedown", onDown);
  }, [open]);

  const toggle = () => {
    const next = !open;
    setOpen(next);
    if (next && unread > 0) markAllRead();
  };

  return (
    <div ref={ref} className="relative no-drag">
      <div className="inline-flex items-center rounded-full bg-white/9 border border-white/11 overflow-hidden shadow-[0_4px_14px_rgba(0,0,0,0.35)]">
        <button
          onClick={toggle}
          title="Notifications"
          className="relative size-10 flex items-center justify-center text-white/55 hover:bg-white/10 hover:text-white/85 transition-colors"
        >
          <Bell className="size-4" />
          {unread > 0 && (
            <span className="absolute top-2 right-2 size-2 rounded-full bg-rose-400 ring-2 ring-[rgba(20,21,28,1)]" />
          )}
        </button>
      </div>

      {open && (
        <div className="absolute right-0 mt-2 w-80 z-50 rounded-2xl overflow-hidden border border-white/10 bg-[rgba(28,28,34,0.85)] backdrop-blur-2xl backdrop-saturate-150 shadow-[0_24px_64px_rgba(0,0,0,0.6),inset_0_1px_0_rgba(255,255,255,0.08)]">
          <div className="flex items-center justify-between px-3 py-2 border-b border-white/[0.06]">
            <span className="text-[12px] font-semibold">Notifications</span>
            {items.length > 0 && (
              <button
                onClick={clear}
                className="inline-flex items-center gap-1 text-[11px] text-[var(--orbit-muted)] hover:text-[var(--orbit-text)]"
              >
                <Trash2 className="size-3" />
                Clear
              </button>
            )}
          </div>
          <div className="max-h-80 overflow-auto scrollbar-thin">
            {items.length === 0 ? (
              <div className="px-3 py-8 text-center text-[12px] text-[var(--orbit-muted)] flex flex-col items-center gap-2">
                <Check className="size-5 opacity-40" />
                No notifications.
              </div>
            ) : (
              items.map((n) => (
                <div key={n.id} className="flex items-start gap-2.5 px-3 py-2.5 border-b border-white/[0.04]">
                  <span className="mt-0.5 shrink-0">{iconFor(n.kind)}</span>
                  <div className="min-w-0 flex-1">
                    <p className="text-[12px] font-medium text-[var(--orbit-text)]">{n.title}</p>
                    <p className="text-[11.5px] text-[var(--orbit-muted)] leading-snug">{n.body}</p>
                    <p className={cn("mt-0.5 text-[10px] text-[var(--orbit-subtle)]")}>{ago(n.ts)}</p>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}
