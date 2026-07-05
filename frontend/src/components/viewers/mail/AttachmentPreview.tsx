import { ExternalLink, X } from "lucide-react";
import { api } from "../../../api";

export function AttachmentPreview({
  url,
  name,
  onClose,
}: {
  url: string;
  name: string;
  onClose: () => void;
}) {
  return (
    <div className="absolute inset-0 z-20 flex flex-col bg-background">
      <div className="flex items-center justify-between gap-4 px-4 py-2.5 border-b border-white/10">
        <span className="text-[12px] font-medium truncate">{name}</span>
        <div className="flex items-center gap-1 shrink-0">
          <button
            className="inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded text-[var(--orbit-muted)] hover:bg-white/[0.06] hover:text-white"
            title="Open in browser"
            onClick={() => api.openURL(url)}
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
      <iframe
        title={name}
        src={url}
        className="flex-1 min-h-0 w-full bg-white"
      />
    </div>
  );
}
