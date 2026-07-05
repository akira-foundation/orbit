import { useState } from "react";
import { useMailbox } from "./useMailbox";
import { MessageList } from "./MessageList";
import { MessageDetail } from "./MessageDetail";

export function MailView() {
  const { messages, selected, loading, error, select, remove } = useMailbox();
  const [selectedId, setSelectedId] = useState<string | null>(null);

  if (error) {
    return (
      <div className="h-full grid place-items-center text-center px-6">
        <div className="space-y-1">
          <p className="text-[13px] font-medium">Mailpit is not available</p>
          <p className="text-[12px] text-[var(--orbit-muted)]">
            Start the Mailpit service to view captured mail.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full flex">
      <div className="w-80 shrink-0 border-r border-white/10 overflow-auto scrollbar-thin">
        {loading ? (
          <div className="p-6 text-[12px] text-[var(--orbit-muted)]">
            Loading...
          </div>
        ) : (
          <MessageList
            messages={messages}
            selectedId={selectedId}
            onSelect={(id) => {
              setSelectedId(id);
              void select(id);
            }}
          />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <MessageDetail
          message={selected}
          onDelete={(id) => {
            setSelectedId(null);
            void remove(id);
          }}
        />
      </div>
    </div>
  );
}
