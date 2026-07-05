import { useState } from "react";
import { Trash2 } from "lucide-react";
import { useMailbox } from "./useMailbox";
import { MessageList } from "./MessageList";
import { MessageDetail } from "./MessageDetail";
import { useProjects } from "../../../store";
import { Button } from "../../ui/button";
import { ConfirmDialog } from "../../ConfirmDialog";

export function MailView() {
  const { messages, selected, unread, loading, error, select, remove, removeAll } =
    useMailbox();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [confirmAll, setConfirmAll] = useState(false);
  const setView = useProjects((s) => s.setView);

  if (error) {
    return (
      <div className="h-full grid place-items-center text-center px-6">
        <div className="space-y-2">
          <p className="text-[13px] font-medium">Mailpit is not running</p>
          <p className="text-[12px] text-[var(--orbit-muted)]">
            Install and start Mailpit to capture and view outgoing mail.
          </p>
          <Button variant="outline" size="sm" onClick={() => setView("services")}>
            Open Services
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full flex">
      <div className="w-80 shrink-0 border-r border-white/10 flex flex-col">
        <div className="flex items-center justify-between px-4 py-2.5 border-b border-white/10">
          <span className="text-[12px] font-medium">
            Inbox
            {unread > 0 && (
              <span className="ml-1.5 text-[var(--orbit-muted)]">{unread} unread</span>
            )}
          </span>
          <Button
            variant="ghost"
            size="sm"
            className="text-rose-300 hover:bg-rose-500/10 hover:text-rose-200"
            disabled={messages.length === 0}
            onClick={() => setConfirmAll(true)}
          >
            <Trash2 className="size-3.5 mr-1.5" />
            Delete all
          </Button>
        </div>
        <div className="flex-1 overflow-auto scrollbar-thin">
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
              onDelete={(id) => setPendingId(id)}
            />
          )}
        </div>
      </div>
      <div className="flex-1 min-w-0">
        <MessageDetail message={selected} />
      </div>

      <ConfirmDialog
        open={pendingId !== null}
        onOpenChange={(open) => !open && setPendingId(null)}
        title="Delete this message?"
        description="This message will be permanently removed from the inbox."
        confirmLabel="Delete"
        destructive
        onConfirm={async () => {
          if (!pendingId) return;
          if (selectedId === pendingId) setSelectedId(null);
          await remove(pendingId);
          setPendingId(null);
        }}
      />

      <ConfirmDialog
        open={confirmAll}
        onOpenChange={setConfirmAll}
        title="Delete all messages?"
        description="Every captured message will be permanently removed from the inbox."
        confirmLabel="Delete all"
        destructive
        onConfirm={async () => {
          setSelectedId(null);
          await removeAll();
        }}
      />
    </div>
  );
}
