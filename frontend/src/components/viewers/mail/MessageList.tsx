import type { mailpit } from "../../../../wailsjs/go/models";
import { cn } from "../../../lib/cn";

type Summary = mailpit.MessageSummary;

export function MessageList({
  messages,
  selectedId,
  onSelect,
}: {
  messages: Summary[];
  selectedId: string | null;
  onSelect: (id: string) => void;
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
        <li key={m.ID}>
          <button
            className={cn(
              "w-full text-left px-4 py-3 hover:bg-white/[0.03]",
              selectedId === m.ID && "bg-white/[0.05]",
            )}
            onClick={() => onSelect(m.ID)}
          >
            <div className="flex items-center gap-2">
              {!m.Read && (
                <span className="size-2 rounded-full bg-[var(--orbit-accent-2)] shrink-0" />
              )}
              <span className="text-[13px] font-medium truncate">
                {m.From?.Name || m.From?.Address || "unknown"}
              </span>
            </div>
            <p className="text-[12px] truncate">{m.Subject || "(no subject)"}</p>
            <p className="text-[11px] text-[var(--orbit-subtle)] truncate">
              {m.Snippet}
            </p>
          </button>
        </li>
      ))}
    </ul>
  );
}
