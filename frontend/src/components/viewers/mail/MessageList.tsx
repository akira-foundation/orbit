import { Trash2 } from "lucide-react";
import type { mailpit } from "../../../../wailsjs/go/models";
import { cn } from "../../../lib/cn";

type Summary = mailpit.MessageSummary;

function receivedAt(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  const now = new Date();
  if (date.toDateString() === now.toDateString()) {
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }
  return date.toLocaleDateString([], { day: "2-digit", month: "short" });
}

export function MessageList({
  messages,
  selectedId,
  onSelect,
  onDelete,
}: {
  messages: Summary[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  if (messages.length === 0) {
    return (
      <div className="p-6 text-[12px] text-[var(--orbit-muted)]">
        No messages captured yet.
      </div>
    );
  }
  return (
    <ul className="divide-y divide-white/5">
      {messages.map((m) => (
        <li
          key={m.ID}
          className={cn(
            "group relative flex items-start hover:bg-white/[0.03]",
            selectedId === m.ID && "bg-white/[0.05]",
          )}
        >
          <button
            className="min-w-0 flex-1 text-left px-4 py-3"
            onClick={() => onSelect(m.ID)}
          >
            <div className="flex items-center gap-2">
              {!m.Read && (
                <span className="size-2 rounded-full bg-[var(--orbit-accent-2)] shrink-0" />
              )}
              <span className="text-[13px] font-medium truncate">
                {m.From?.Name || m.From?.Address || "unknown"}
              </span>
              <span className="ml-auto shrink-0 text-[10.5px] text-[var(--orbit-subtle)] group-hover:opacity-0">
                {receivedAt(m.Created)}
              </span>
            </div>
            <p className="text-[12px] truncate">{m.Subject || "(no subject)"}</p>
            <p className="text-[11px] text-[var(--orbit-subtle)] truncate">
              {m.Snippet}
            </p>
          </button>
          <button
            className="absolute right-2 top-2 hidden rounded p-1.5 text-rose-300 hover:bg-rose-500/10 hover:text-rose-200 group-hover:block"
            title="Delete message"
            onClick={() => onDelete(m.ID)}
          >
            <Trash2 className="size-3.5" />
          </button>
        </li>
      ))}
    </ul>
  );
}
