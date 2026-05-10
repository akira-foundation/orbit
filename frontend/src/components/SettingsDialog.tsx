import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "./ui/dialog";
import { Button } from "./ui/button";
import { Switch } from "./ui/switch";
import {
  Globe,
  Info,
  Orbit,
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
  description?: React.ReactNode;
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
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = async () => {
    try {
      setStatus(await api.systemStatus());
    } catch (e: any) {
      setError(e?.message ?? String(e));
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const toggle = async (next: boolean) => {
    setBusy(true);
    setError(null);
    try {
      await api.setLaunchAtLogin(next);
      await refresh();
    } catch (e: any) {
      setError(e?.message ?? String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <SectionHeader title="General" description="Application preferences." />
      <div className="px-6 py-5 space-y-4">
        <ToggleRow
          label="Start Orbit at login"
          description="Open Orbit silently in the background when you log in, so *.orbit.test sites work without thinking about it."
          checked={!!status?.launchAtLogin}
          disabled={busy || !status}
          onChange={toggle}
        />
        {error && (
          <p className="text-[11px] text-rose-300 font-mono whitespace-pre-wrap">
            {error}
          </p>
        )}
      </div>
    </>
  );
}

function ToggleRow({
  label,
  description,
  checked,
  disabled,
  onChange,
}: {
  label: string;
  description?: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (next: boolean) => void;
}) {
  return (
    <div className="flex items-start justify-between gap-4 rounded-lg border border-white/10 bg-white/[0.03] p-4">
      <div className="min-w-0 space-y-1">
        <p className="text-[13px] font-medium">{label}</p>
        {description && (
          <p className="text-[11px] text-[var(--orbit-muted)] leading-relaxed">
            {description}
          </p>
        )}
      </div>
      <Switch
        checked={checked}
        disabled={disabled}
        onCheckedChange={onChange}
        className="mt-0.5 shrink-0"
      />
    </div>
  );
}

function AboutSection() {
  return (
    <>
      <SectionHeader title="About" />

      <div className="px-6 py-10 flex flex-col items-center text-center">
        <Orbit
          className="size-20 text-[var(--orbit-accent)]"
          strokeWidth={1.25}
        />
        <h2 className="mt-5 text-2xl font-semibold tracking-tight">Orbit</h2>
        <p className="mt-1 text-[11px] font-mono uppercase tracking-[0.18em] text-[var(--orbit-muted)]">
          Version 0.1.0
        </p>
        <p className="mt-5 max-w-xs text-[12px] text-[var(--orbit-muted)] leading-relaxed">
          An ambient runtime layer for local web development.
        </p>

        <div className="mt-8 w-full max-w-sm rounded-lg border border-white/[0.06] bg-white/[0.02] divide-y divide-white/[0.04] text-[12px]">
          <InfoRow k="Platform" v="macOS" />
          <InfoRow k="Domain suffix" v="*.orbit.test" />
          <InfoRow k="Proxy" v="127.0.0.2:80" />
        </div>

        <p className="mt-8 text-[11px] text-[var(--orbit-muted)]">
          <span className="text-[var(--orbit-text)]">kidiatoliny</span>
          <span className="mx-1.5 text-[var(--orbit-subtle)]">@</span>
          <span className="text-[var(--orbit-text)]">Akira Foundation</span>
        </p>
      </div>
    </>
  );
}

function InfoRow({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex items-center justify-between px-4 py-2">
      <span className="text-[var(--orbit-muted)]">{k}</span>
      <span className="font-mono text-[var(--orbit-text)]">{v}</span>
    </div>
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
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = async () => {
    setRefreshing(true);
    try {
      setStatus(await api.systemStatus());
    } catch (e: any) {
      setError(e?.message ?? String(e));
    } finally {
      // Keep spinner visible briefly so user sees feedback even on instant
      // returns.
      setTimeout(() => setRefreshing(false), 350);
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
    <div className="flex flex-col h-full">
      <SectionHeader
        title="Local Domains"
        description={
          <>
            <code className="font-mono">orbit-proxyd</code> serves{" "}
            <code className="font-mono">*.orbit.test</code> on 127.0.0.2,
            coexisting with Herd's <code className="font-mono">*.test</code> on
            127.0.0.1.
          </>
        }
      />

      <div className="flex-1 overflow-auto px-6 py-5 space-y-5">
        {/* Overall status banner */}
        <StatusBanner status={status} />

        {/* Detailed checks list */}
        <div className="rounded-lg border border-white/10 bg-white/[0.02] overflow-hidden">
          <div className="flex items-center justify-between px-4 py-2.5 border-b border-white/[0.06]">
            <p className="text-[11px] font-semibold uppercase tracking-widest text-[var(--orbit-subtle)]">
              System checks
            </p>
            <button
              type="button"
              onClick={refresh}
              disabled={!!busy || refreshing}
              className="size-6 inline-flex items-center justify-center rounded text-[var(--orbit-muted)] hover:text-[var(--orbit-text)] hover:bg-white/5 disabled:opacity-50"
              title="Re-check"
            >
              <RotateCcw
                className={cn("size-3.5", refreshing && "animate-spin")}
              />
            </button>
          </div>
          {status ? (
            <CheckList status={status} />
          ) : (
            <div className="px-4 py-6 text-[12px] text-[var(--orbit-muted)] italic text-center">
              Loading…
            </div>
          )}
        </div>

        {error && (
          <pre className="text-[11px] text-rose-300 font-mono whitespace-pre-wrap rounded-md bg-rose-500/5 border border-rose-500/20 p-3">
            {error}
          </pre>
        )}
      </div>

      <footer className="flex items-center justify-between gap-2 px-6 py-3 border-t border-white/[0.06] bg-white/[0.02]">
        <p className="text-[10px] text-[var(--orbit-muted)] flex-1 min-w-0 truncate">
          {status?.setup
            ? "Already configured. Re-run if you reset DNS / loopback."
            : "Requires admin password once."}
        </p>
        <Button
          variant="ghost"
          size="sm"
          onClick={runReset}
          disabled={!!busy || !status?.setup}
        >
          <Trash2 className="size-3.5 mr-1.5" />
          {busy === "reset" ? "Resetting…" : "Reset"}
        </Button>
        <Button size="sm" onClick={runSetup} disabled={!!busy}>
          <ShieldCheck className="size-3.5 mr-1.5" />
          {busy === "setup"
            ? "Setting up…"
            : status?.setup
              ? "Re-run setup"
              : "Set up"}
        </Button>
      </footer>
    </div>
  );
}

function StatusBanner({ status }: { status: SystemStatus | null }) {
  if (!status) return null;
  const ok = status.setup;
  const conflict = status.herdConflict;
  const tone = conflict
    ? {
        ring: "border-rose-400/30 bg-rose-400/8",
        dot: "bg-rose-400 shadow-[0_0_10px_#fb7185]",
        title: "Conflict detected",
        text: status.message,
        textColor: "text-rose-200",
      }
    : ok
      ? {
          ring: "border-emerald-400/30 bg-emerald-400/8",
          dot: "bg-emerald-400 shadow-[0_0_10px_#34d399]",
          title: "Local domains ready",
          text: status.message,
          textColor: "text-emerald-200/80",
        }
      : {
          ring: "border-amber-400/30 bg-amber-400/8",
          dot: "bg-amber-400 shadow-[0_0_10px_#fbbf24]",
          title: "Setup needed",
          text: status.message,
          textColor: "text-amber-200/80",
        };
  return (
    <div
      className={cn(
        "rounded-lg border px-4 py-3 flex items-start gap-3",
        tone.ring,
      )}
    >
      <span
        className={cn("mt-1 size-2 rounded-full shrink-0", tone.dot)}
      />
      <div className="min-w-0 flex-1">
        <p className="text-[13px] font-semibold">{tone.title}</p>
        <p className={cn("text-[11px] mt-0.5", tone.textColor)}>{tone.text}</p>
      </div>
    </div>
  );
}

function CheckList({ status }: { status: SystemStatus }) {
  const items: {
    label: string;
    hint: string;
    ok: boolean;
  }[] = [
    {
      label: "Loopback alias 127.0.0.2",
      hint: "Extra IP on lo0 so proxyd can listen without colliding with 127.0.0.1.",
      ok: status.loopbackOk,
    },
    {
      label: "DNS responder *.orbit.test",
      hint: "Custom DNS on 127.0.0.2:53 answering the orbit subdomain only.",
      ok: status.dnsmasqOk,
    },
    {
      label: "/etc/resolver/orbit.test",
      hint: "macOS resolver entry routing the suffix to the local DNS responder.",
      ok: status.resolverOk,
    },
    {
      label: "Proxy daemon",
      hint: "orbit-proxyd LaunchDaemon listening on 127.0.0.2:80.",
      ok: status.daemonOk,
    },
  ];
  return (
    <ul className="divide-y divide-white/[0.04]">
      {items.map((it) => (
        <li
          key={it.label}
          className="flex items-start gap-3 px-4 py-2.5"
        >
          <span
            className={cn(
              "mt-1 size-2 rounded-full shrink-0",
              it.ok
                ? "bg-emerald-400 shadow-[0_0_8px_#34d399]"
                : "bg-amber-400 shadow-[0_0_8px_#fbbf24]",
            )}
          />
          <div className="min-w-0 flex-1">
            <p className="text-[12px] font-medium text-[var(--orbit-text)]">
              {it.label}
            </p>
            <p className="text-[10px] text-[var(--orbit-muted)] leading-relaxed">
              {it.hint}
            </p>
          </div>
          <span
            className={cn(
              "text-[10px] font-mono uppercase tracking-widest shrink-0",
              it.ok ? "text-emerald-300" : "text-amber-300",
            )}
          >
            {it.ok ? "OK" : "Pending"}
          </span>
        </li>
      ))}
    </ul>
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
