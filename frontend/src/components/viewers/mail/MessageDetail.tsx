import { useEffect, useState } from "react";
import { Paperclip } from "lucide-react";
import type { mailpit } from "../../../../wailsjs/go/models";
import { AttachmentPreview } from "./AttachmentPreview";

type Message = mailpit.Message;

function partURL(messageId: string, partId: string): string {
  return `http://mail.orbit.test/api/v1/message/${encodeURIComponent(messageId)}/part/${encodeURIComponent(partId)}`;
}

function MessageBody({ html, text }: { html: string; text: string }) {
  const [url, setUrl] = useState<string | null>(null);

  useEffect(() => {
    if (!html) {
      setUrl(null);
      return;
    }
    const objectUrl = URL.createObjectURL(
      new Blob([html], { type: "text/html" }),
    );
    setUrl(objectUrl);
    return () => URL.revokeObjectURL(objectUrl);
  }, [html]);

  if (!html) {
    return <pre className="text-[12px] whitespace-pre-wrap">{text}</pre>;
  }
  if (!url) return null;
  return (
    <iframe
      title="message"
      sandbox="allow-same-origin"
      src={url}
      className="w-full h-full bg-white rounded"
    />
  );
}

export function MessageDetail({ message }: { message: Message | null }) {
  const [preview, setPreview] = useState<{
    partId: string;
    name: string;
  } | null>(null);

  const messageId = message?.ID;
  useEffect(() => {
    setPreview(null);
  }, [messageId]);

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
        <MessageBody html={message.HTML} text={message.Text} />
      </div>
      {(message.Attachments ?? []).length > 0 && (
        <div className="px-5 py-3 border-t border-white/10 flex flex-wrap gap-2">
          {message.Attachments.map((a) => (
            <button
              key={a.PartID}
              className="inline-flex items-center gap-1.5 text-[11px] px-2 py-1 rounded bg-white/[0.05] hover:bg-white/[0.09] text-[var(--orbit-accent-2)]"
              title={`Preview ${a.FileName}`}
              onClick={() =>
                setPreview({ partId: a.PartID, name: a.FileName })
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
          messageId={message.ID}
          partId={preview.partId}
          name={preview.name}
          externalUrl={partURL(message.ID, preview.partId)}
          onClose={() => setPreview(null)}
        />
      )}
    </div>
  );
}
