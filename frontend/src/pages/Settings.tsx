import { useEffect, useState } from "react";
import {
  SETTINGS_SECTIONS,
  SettingsSection,
  type SettingsSectionID,
} from "../components/SettingsSections";
import { cn } from "../lib/cn";

export function SettingsPage({
  autoAction,
  onAutoActionConsumed,
}: {
  autoAction?: "setup" | "reset" | null;
  onAutoActionConsumed?: () => void;
}) {
  const [section, setSection] = useState<SettingsSectionID>("general");

  useEffect(() => {
    if (autoAction) setSection("domains");
  }, [autoAction]);

  return (
    <div className="h-full overflow-auto scrollbar-thin">
      <div className="mx-auto w-full max-w-3xl px-8 py-10 space-y-5">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
          <p className="text-[12px] text-[var(--orbit-muted)] mt-1">
            Application preferences, local domains and bundled services.
          </p>
        </div>

        <div className="flex items-center gap-1 rounded-lg border border-[var(--orbit-border)] bg-white/[0.02] p-1 w-fit">
          {SETTINGS_SECTIONS.map((s) => (
            <button
              key={s.id}
              onClick={() => setSection(s.id)}
              className={cn(
                "h-7 px-3 rounded-md inline-flex items-center gap-1.5 text-[12px] transition",
                section === s.id
                  ? "bg-white/[0.10] text-white"
                  : "text-white/70 hover:bg-white/[0.05]",
              )}
            >
              <s.icon className="size-3.5 shrink-0 text-[var(--orbit-muted)]" />
              {s.label}
            </button>
          ))}
        </div>

        <div className="rounded-2xl border border-[var(--orbit-border)] bg-white/2 overflow-hidden">
          <SettingsSection
            section={section}
            autoAction={autoAction}
            onAutoActionConsumed={onAutoActionConsumed}
            visible
          />
        </div>
      </div>
    </div>
  );
}
