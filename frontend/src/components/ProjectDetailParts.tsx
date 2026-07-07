import { Lock, LockOpen } from "lucide-react";
import { Button } from "./ui/button";
import { cn } from "../lib/cn";

export function Cell({
  icon,
  label,
  value,
  accent,
  ok,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  accent?: boolean;
  ok?: boolean;
}) {
  const valCls = ok
    ? "text-emerald-300"
    : accent
      ? "text-[var(--orbit-accent-2)]"
      : "text-[var(--orbit-text)]";
  return (
    <div className="px-6 py-4 min-w-0">
      <div className="flex items-center gap-1.5 text-[11px] text-[var(--orbit-muted)]">
        <span className="text-[var(--orbit-subtle)]">{icon}</span>
        {label}
      </div>
      <div className={cn("mt-1 text-[14px] font-mono truncate", valCls)}>
        {value}
      </div>
    </div>
  );
}

export function Card({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 p-6 space-y-5">
      <header className="space-y-0.5">
        <h2 className="text-[15px] font-semibold tracking-tight">{title}</h2>
        {subtitle && (
          <p className="text-[11.5px] text-[var(--orbit-muted)] leading-relaxed">
            {subtitle}
          </p>
        )}
      </header>
      {children}
    </section>
  );
}

export function Detail({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="grid grid-cols-[7rem_1fr] items-center gap-3 py-3 first:pt-0 last:pb-0">
      <dt className="text-[12px] text-[var(--orbit-muted)]">{label}</dt>
      <dd className="min-w-0 text-right">{children}</dd>
    </div>
  );
}

export function SecureToggle({
  projectId,
  secure,
  onChange,
}: {
  projectId: string;
  secure: boolean;
  onChange: (next: boolean) => void;
}) {
  void projectId;
  return (
    <button
      onClick={() => onChange(!secure)}
      title={secure ? "HTTPS only — click to disable" : "Force HTTPS"}
      className={cn(
        "inline-flex items-center gap-1.5 h-6 px-2 rounded-md border text-[10.5px] font-medium tracking-wide transition-colors",
        secure
          ? "border-emerald-400/30 bg-emerald-400/10 text-emerald-300"
          : "border-white/10 bg-white/[0.03] text-[var(--orbit-muted)] hover:text-[var(--orbit-text)]",
      )}
    >
      {secure ? <Lock className="size-3" /> : <LockOpen className="size-3" />}
      {secure ? "HTTPS" : "HTTP"}
    </button>
  );
}

export function IconBtn({
  icon,
  onClick,
  disabled,
  title,
}: {
  icon: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  title: string;
}) {
  return (
    <Button variant="ghost" size="icon" onClick={onClick} disabled={disabled} title={title}>
      {icon}
    </Button>
  );
}
