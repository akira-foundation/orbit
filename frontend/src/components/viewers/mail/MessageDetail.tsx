import { useState } from "react";
import { Paperclip } from "lucide-react";
import type { mailpit } from "../../../../wailsjs/go/models";
import { AttachmentPreview } from "./AttachmentPreview";

type Message = mailpit.Message;

function partURL(messageId: string, partId: string): string {
  return `http://mail.orbit.test/api/v1/message/${encodeURIComponent(messageId)}/part/${encodeURIComponent(partId)}`;
}

export function MessageDetail({ message }: { message: Message | null }) {
  const [preview, setPreview] = useState<{ url: string; name: string } | null>(
    null,
  );

  if (!message) {
    return (
      <div className="h-full grid place-items-center text-[12px] text-[var(--orbit-muted)]">
        Select a message
      </div>
    );
  }
  return (
    <div className="relative h-full flex flex-col">
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
            <button
              key={a.PartID}
              className="inline-flex items-center gap-1.5 text-[11px] px-2 py-1 rounded bg-white/[0.05] hover:bg-white/[0.09] text-[var(--orbit-accent-2)]"
              title={`Preview ${a.FileName}`}
              onClick={() =>
                setPreview({
                  url: partURL(message.ID, a.PartID),
                  name: a.FileName,
                })
              }
            >
              <Paperclip className="size-3" />
              {a.FileName}
            </button>
          ))}
        </div>
      )}
      {preview && (
        <AttachmentPreview
          url={preview.url}
          name={preview.name}
          onClose={() => setPreview(null)}
        />
      )}
    </div>
  );
}
