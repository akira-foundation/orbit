import { useEffect, useState } from "react";
import { Check, Copy, Inbox } from "lucide-react";
import { useCopyToClipboard } from "usehooks-ts";
import { api } from "../../../api";
import type { ServiceSetupSnippet } from "../../../types";

export function MailEmptyState() {
  const [snippet, setSnippet] = useState<ServiceSetupSnippet | null>(null);
  const [copied, setCopied] = useState(false);
  const [, copy] = useCopyToClipboard();

  useEffect(() => {
    api
      .serviceSetup("mailpit")
      .then((setup) => {
        const env = setup.snippets.find((s) => s.label === ".env");
        setSnippet(env ?? setup.snippets[0] ?? null);
      })
      .catch(() => setSnippet(null));
  }, []);

  return (
    <div className="h-full grid place-items-center text-center px-6">
      <div className="w-full max-w-sm space-y-4">
        <div className="mx-auto size-12 rounded-full bg-white/[0.04] border border-white/10 grid place-items-center">
          <Inbox className="size-5 text-[var(--orbit-muted)]" />
        </div>
        <div className="space-y-1.5">
          <p className="text-[14px] font-medium">Your inbox is empty</p>
          <p className="text-[12px] text-[var(--orbit-muted)] leading-relaxed">
            Mailpit captures every email your apps send. Point your app's mail
            transport at these settings and messages appear here instantly.
          </p>
        </div>
        {snippet && (
          <div className="rounded-lg border border-white/10 bg-black/30 text-left overflow-hidden">
            <div className="flex items-center justify-between px-3 py-1.5 border-b border-white/10">
              <span className="text-[10.5px] uppercase tracking-wide text-[var(--orbit-subtle)]">
                {snippet.label}
              </span>
              <button
                className="inline-flex items-center gap-1 text-[11px] text-[var(--orbit-muted)] hover:text-white"
                onClick={() => {
                  void copy(snippet.code);
                  setCopied(true);
                  window.setTimeout(() => setCopied(false), 1500);
                }}
              >
                {copied ? (
                  <Check className="size-3" />
                ) : (
                  <Copy className="size-3" />
                )}
                {copied ? "Copied" : "Copy"}
              </button>
            </div>
            <pre className="px-3 py-2.5 text-[11.5px] leading-relaxed whitespace-pre-wrap">
              {snippet.code}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
}
