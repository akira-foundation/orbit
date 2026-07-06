import { useEffect, useState } from "react";
import { ExternalLink, X } from "lucide-react";
import { api } from "../../../api";
import { PdfViewer } from "./PdfViewer";

function toBytes(data: string | number[]): Uint8Array {
  if (typeof data === "string") {
    const bin = atob(data);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) arr[i] = bin.charCodeAt(i);
    return arr;
  }
  return Uint8Array.from(data);
}

export function AttachmentPreview({
  messageId,
  partId,
  name,
  externalUrl,
  onClose,
}: {
  messageId: string;
  partId: string;
  name: string;
  externalUrl: string;
  onClose: () => void;
}) {
  const [bytes, setBytes] = useState<Uint8Array | null>(null);
  const [contentType, setContentType] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setBytes(null);
    setError(null);
    api
      .mailPart(messageId, partId)
      .then((part) => {
        if (cancelled) return;
        setContentType(part.contentType);
        setBytes(toBytes(part.data as unknown as string | number[]));
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [messageId, partId]);

  return (
    <div className="absolute inset-0 z-20 flex flex-col bg-background">
      <div className="flex items-center justify-between gap-4 px-4 py-2.5 border-b border-white/10">
        <span className="text-[12px] font-medium truncate">{name}</span>
        <div className="flex items-center gap-1 shrink-0">
          <button
            className="inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded text-[var(--orbit-muted)] hover:bg-white/[0.06] hover:text-white"
            title="Open in browser"
            onClick={() => api.openURL(externalUrl)}
          >
            <ExternalLink className="size-3.5" />
          </button>
          <button
            className="inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded text-[var(--orbit-muted)] hover:bg-white/[0.06] hover:text-white"
            title="Close"
            onClick={onClose}
          >
            <X className="size-4" />
          </button>
        </div>
      </div>
      <div className="flex-1 min-h-0">
        <PreviewBody
          bytes={bytes}
          contentType={contentType}
          name={name}
          error={error}
        />
      </div>
    </div>
  );
}

function PreviewBody({
  bytes,
  contentType,
  name,
  error,
}: {
  bytes: Uint8Array | null;
  contentType: string;
  name: string;
  error: string | null;
}) {
  const [imageUrl, setImageUrl] = useState<string | null>(null);
  const isImage = contentType.startsWith("image/");

  useEffect(() => {
    if (!bytes || !isImage) return;
    const url = URL.createObjectURL(
      new Blob([bytes as BlobPart], { type: contentType }),
    );
    setImageUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [bytes, isImage, contentType]);

  if (error) {
    return (
      <div className="h-full grid place-items-center text-[12px] text-[var(--orbit-muted)]">
        {error}
      </div>
    );
  }
  if (!bytes) {
    return (
      <div className="h-full grid place-items-center text-[12px] text-[var(--orbit-muted)]">
        Loading...
      </div>
    );
  }
  if (contentType.includes("pdf")) {
    return <PdfViewer data={bytes} />;
  }
  if (isImage && imageUrl) {
    return (
      <div className="h-full overflow-auto scrollbar-thin bg-neutral-800 grid place-items-center p-4">
        <img src={imageUrl} alt={name} className="max-w-full h-auto" />
      </div>
    );
  }
  return (
    <div className="h-full grid place-items-center text-center text-[12px] text-[var(--orbit-muted)] px-6">
      No inline preview for this file type. Use the open button to view it in
      your browser.
    </div>
  );
}
