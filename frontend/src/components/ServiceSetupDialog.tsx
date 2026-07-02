import { useEffect, useState } from "react";
import { useCopyToClipboard } from "usehooks-ts";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { Check, Copy } from "lucide-react";
import { api } from "../api";
import type { ServiceSetup } from "../types";
import { cn } from "../lib/cn";

interface Props {
  engine: string;
  displayName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ServiceSetupDialog({
  engine,
  displayName,
  open,
  onOpenChange,
}: Props) {
  const [setup, setSetup] = useState<ServiceSetup | null>(null);
  const [, copy] = useCopyToClipboard();
  const [copied, setCopied] = useState<string | null>(null);

  useEffect(() => {
    if (open) api.serviceSetup(engine).then(setSetup);
  }, [open, engine]);

  async function copyCode(label: string, code: string) {
    await copy(code);
    setCopied(label);
    setTimeout(() => setCopied(null), 1500);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg p-0 sm:rounded-2xl overflow-hidden">
        <DialogHeader className="px-6 pt-6 pb-4 border-b border-white/[0.06] space-y-0.5">
          <DialogTitle className="text-[15px]">
            Connect {displayName}
          </DialogTitle>
          <DialogDescription className="text-[12px] text-[var(--orbit-muted)]">
            Point your project at the local {displayName} instance.
          </DialogDescription>
        </DialogHeader>

        <div className="px-6 py-5 space-y-5 max-h-[60vh] overflow-auto scrollbar-thin">
          {setup ? (
            <>
              <dl className="rounded-lg border border-white/[0.06] bg-white/[0.02] divide-y divide-white/[0.04] text-[12px]">
                {setup.fields.map((f) => (
                  <div
                    key={f.label}
                    className="flex items-center justify-between px-4 py-2.5"
                  >
                    <dt className="text-[var(--orbit-muted)]">{f.label}</dt>
                    <dd className="font-mono text-[var(--orbit-text)]">
                      {f.value}
                    </dd>
                  </div>
                ))}
              </dl>

              {setup.snippets.map((s) => (
                <div key={s.label} className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--orbit-subtle)]">
                      {s.label}
                    </span>
                    <button
                      className="inline-flex items-center gap-1 text-[11px] text-[var(--orbit-muted)] hover:text-[var(--orbit-text)] transition-colors"
                      onClick={() => copyCode(s.label, s.code)}
                    >
                      {copied === s.label ? (
                        <Check className="size-3.5" />
                      ) : (
                        <Copy className="size-3.5" />
                      )}
                      {copied === s.label ? "Copied" : "Copy"}
                    </button>
                  </div>
                  <pre
                    className={cn(
                      "rounded-lg border border-white/[0.06] bg-black/30 p-3",
                      "text-[11.5px] font-mono leading-relaxed text-[var(--orbit-text)]",
                      "overflow-x-auto whitespace-pre scrollbar-thin",
                    )}
                  >
                    {s.code}
                  </pre>
                </div>
              ))}
            </>
          ) : (
            <p className="text-[12px] text-[var(--orbit-muted)] italic text-center py-6">
              Loading…
            </p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
