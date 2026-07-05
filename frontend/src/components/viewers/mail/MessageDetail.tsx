import type { mailpit } from "../../../../wailsjs/go/models";

type Message = mailpit.Message;

export function MessageDetail({ message }: { message: Message | null }) {
  if (!message) {
    return (
      <div className="h-full grid place-items-center text-[12px] text-[var(--orbit-muted)]">
        Select a message
      </div>
    );
  }
  return (
    <div className="h-full flex flex-col">
      <div className="px-5 py-4 border-b border-white/10 space-y-1">
        <h2 className="text-[14px] font-semibold">
          {message.Subject || "(no subject)"}
        </h2>
        <p className="text-[11px] text-[var(--orbit-muted)]">
          {message.From?.Address} to{" "}
          {(message.To ?? []).map((t) => t.Address).join(", ")}
        </p>
      </div>
      <div className="flex-1 min-h-0 overflow-auto scrollbar-thin p-5">
        {message.HTML ? (
          <iframe
            title="message"
            sandbox=""
            srcDoc={message.HTML}
            className="w-full h-full bg-white rounded"
          />
        ) : (
          <pre className="text-[12px] whitespace-pre-wrap">{message.Text}</pre>
        )}
      </div>
      {(message.Attachments ?? []).length > 0 && (
        <div className="px-5 py-3 border-t border-white/10 flex flex-wrap gap-2">
          {message.Attachments.map((a) => (
            <span
              key={a.PartID}
              className="text-[11px] px-2 py-1 rounded bg-white/[0.05]"
            >
              {a.FileName}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
