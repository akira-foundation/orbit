import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "./ui/dialog";
import { Button } from "./ui/button";
import {
  Globe,
  Info,
  RotateCcw,
  Settings as SettingsIcon,
  ShieldCheck,
  Trash2,
} from "lucide-react";
import { api } from "../api";
import type { SystemStatus } from "../types";
import { cn } from "../lib/cn";

type Section = "general" | "domains" | "about";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialSection?: Section;
  autoAction?: "setup" | "reset" | null;
  onAutoActionConsumed?: () => void;
}

const SECTIONS: { id: Section; label: string; icon: typeof SettingsIcon }[] = [
  { id: "general", label: "General", icon: SettingsIcon },
  { id: "domains", label: "Local Domains", icon: Globe },
  { id: "about", label: "About", icon: Info },
];

export function SettingsDialog({
  open,
  onOpenChange,
  initialSection = "general",
  autoAction,
  onAutoActionConsumed,
}: Props) {
  const [section, setSection] = useState<Section>(initialSection);

  useEffect(() => {
    if (open) setSection(autoAction ? "domains" : initialSection);
  }, [open, autoAction, initialSection]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl p-0 sm:rounded-2xl">
        <DialogTitle className="sr-only">Settings</DialogTitle>
        <div className="flex h-[520px]">
          <aside className="w-52 shrink-0 border-r border-white/[0.06] bg-white/[0.02] p-3">
            <p className="px-2 pt-1 pb-2 text-[11px] font-semibold uppercase tracking-wider text-[var(--orbit-subtle)]">
              Settings
            </p>
            <nav className="space-y-0.5">
              {SECTIONS.map((s) => (
                <button
                  key={s.id}
                  onClick={() => setSection(s.id)}
                  className={cn(
                    "w-full h-8 px-2.5 rounded-md flex items-center gap-2 text-[13px] transition",
                    section === s.id
                      ? "bg-white/[0.10] text-white"
                      : "text-white/75 hover:bg-white/[0.05]",
                  )}
                >
                  <s.icon className="size-3.5 shrink-0 text-[var(--orbit-muted)]" />
                  {s.label}
                </button>
              ))}
            </nav>
          </aside>

          <div className="flex-1 min-w-0 overflow-auto">
            {section === "general" && <GeneralSection />}
            {section === "domains" && (
              <LocalDomainsSection
                autoAction={autoAction}
                onAutoActionConsumed={onAutoActionConsumed}
                visible={open}
              />
            )}
            {section === "about" && <AboutSection />}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function SectionHeader({
  title,
  description,
}: {
  title: string;
  description?: string;
}) {
  return (
    <div className="px-6 pt-6 pb-4 border-b border-white/[0.06]">
      <h2 className="text-[15px] font-semibold">{title}</h2>
      {description && (
        <p className="text-[12px] text-[var(--orbit-muted)] mt-0.5">
          {description}
        </p>
      )}
    </div>
  );
}

function GeneralSection() {
  return (
    <>
      <SectionHeader title="General" description="Application preferences." />
      <div className="px-6 py-6 text-[13px] text-[var(--orbit-muted)]">
        Nothing here yet.
      </div>
    </>
  );
}

function AboutSection() {
  return (
    <>
      <SectionHeader title="About" />
      <div className="px-6 py-6 space-y-1 text-[13px]">
        <p className="font-semibold">Orbit</p>
        <p className="text-[var(--orbit-muted)]">Smart Runtime Orchestration</p>
        <p className="text-[var(--orbit-muted)] pt-2 text-[11px]">
          © Akira Foundation
        </p>
      </div>
    </>
  );
}

function LocalDomainsSection({
  autoAction,
  onAutoActionConsumed,
  visible,
}: {
  autoAction?: "setup" | "reset" | null;
  onAutoActionConsumed?: () => void;
  visible: boolean;
}) {
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [busy, setBusy] = useState<"setup" | "reset" | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = async () => {
    try {
      setStatus(await api.systemStatus());
    } catch (e: any) {
      setError(e?.message ?? String(e));
    }
  };

  const runSetup = async () => {
    setBusy("setup");
    setError(null);
    try {
      await api.systemSetup();
      await refresh();
    } catch (e: any) {
      setError(e?.message ?? String(e));
    } finally {
      setBusy(null);
    }
  };

  const runReset = async () => {
    setBusy("reset");
    setError(null);
    try {
      await api.systemUninstall();
      await refresh();
    } catch (e: any) {
      setError(e?.message ?? String(e));
    } finally {
      setBusy(null);
    }
  };

  useEffect(() => {
    if (visible) {
      setError(null);
      refresh();
    }
  }, [visible]);

  useEffect(() => {
    if (!visible || !autoAction) return;
    if (autoAction === "setup") runSetup();
    else if (autoAction === "reset") runReset();
    onAutoActionConsumed?.();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, autoAction]);

  return (
    <>
      <SectionHeader
        title="Local Domains"
        description="orbit-proxyd serves *.orbit.test on 127.0.0.2 — coexists with Herd's *.test on 127.0.0.1."
      />
      <div className="px-6 py-5 space-y-4">
        <div className="rounded-lg border border-white/10 bg-white/[0.03] p-4 space-y-3">
          <div className="flex items-center justify-between">
            <p className="text-[13px] font-semibold">System status</p>
            <button
              onClick={refresh}
              disabled={!!busy}
              className="size-7 inline-flex items-center justify-center rounded-md border border-white/10 hover:bg-white/5 text-white/70"
              title="Re-check"
            >
              <RotateCcw className="size-3.5" />
            </button>
          </div>
          {status && <Checks status={status} />}
          {status && (
            <p
              className={cn(
                "text-[12px]",
                status.setup ? "text-emerald-300" : "text-amber-200/80",
              )}
            >
              {status.message}
            </p>
          )}
        </div>

        {error && (
          <pre className="text-[11px] text-rose-300 font-mono whitespace-pre-wrap rounded-md bg-rose-500/5 border border-rose-500/20 p-3">
            {error}
          </pre>
        )}

        <div className="flex items-center justify-end gap-2 pt-1">
          <Button variant="ghost" onClick={runReset} disabled={!!busy}>
            <Trash2 className="size-3.5 mr-1.5" />
            {busy === "reset" ? "Resetting…" : "Reset"}
          </Button>
          <Button onClick={runSetup} disabled={!!busy}>
            <ShieldCheck className="size-3.5 mr-1.5" />
            {busy === "setup"
              ? "Setting up…"
              : status?.setup
                ? "Re-run setup"
                : "Set up"}
          </Button>
        </div>
      </div>
    </>
  );
}

function Checks({ status }: { status: SystemStatus }) {
  const items: [string, boolean][] = [
    ["Loopback 127.0.0.2", status.loopbackOk],
    ["DNS *.orbit.test", status.dnsmasqOk],
    ["resolver", status.resolverOk],
    ["proxy daemon", status.daemonOk],
  ];
  return (
    <div className="flex flex-wrap gap-1.5">
      {items.map(([label, ok]) => (
        <span
          key={label}
          className={cn(
            "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium",
            ok
              ? "bg-emerald-400/10 text-emerald-300 border border-emerald-400/20"
              : "bg-amber-400/10 text-amber-200 border border-amber-400/25",
          )}
        >
          <span
            className={cn(
              "size-1.5 rounded-full",
              ok ? "bg-emerald-400" : "bg-amber-400",
            )}
          />
          {label}
        </span>
      ))}
    </div>
  );
}
