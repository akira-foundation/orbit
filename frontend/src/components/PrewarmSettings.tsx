import { useEffect, useState } from "react";
import { api } from "../api";
import type { RuntimesConfig } from "../types";
import { cn } from "../lib/cn";
import { ToggleRow } from "./SettingsSections";

const HOVER_PRESETS = [500, 1000, 2000, 3000];

export function PrewarmSettings() {
  const [cfg, setCfg] = useState<RuntimesConfig | null>(null);

  useEffect(() => {
    api.runtimesConfig().then(setCfg);
  }, []);

  const save = async (next: RuntimesConfig) => {
    setCfg(next);
    await api.saveRuntimesConfig(next);
  };

  return (
    <>
      <ToggleRow
        label="Smart prewarm"
        description="Start a project's runtime ahead of the first request, on hover and from past usage patterns, so opening it is instant."
        checked={!!cfg && cfg.prewarmEnabled}
        disabled={!cfg}
        onChange={(v) => cfg && save({ ...cfg, prewarmEnabled: v })}
      />
      {cfg?.prewarmEnabled ? (
        <div className="flex items-center justify-between gap-4 rounded-lg border border-white/10 bg-white/[0.03] p-4">
          <div className="min-w-0 space-y-1">
            <p className="text-[13px] font-medium">Hover delay</p>
            <p className="text-[11px] text-[var(--orbit-muted)] leading-relaxed">
              How long to rest the pointer on a project before it prewarms.
            </p>
          </div>
          <div className="flex items-center gap-1 shrink-0">
            {HOVER_PRESETS.map((ms) => {
              const active = (cfg.prewarmHoverMs || 1000) === ms;
              return (
                <button
                  key={ms}
                  onClick={() => save({ ...cfg, prewarmHoverMs: ms })}
                  className={cn(
                    "h-7 px-2.5 rounded-md text-[12px] transition",
                    active
                      ? "bg-[var(--orbit-accent-2)]/20 text-[var(--orbit-accent-2)] border border-[var(--orbit-accent-2)]/40"
                      : "text-[var(--orbit-muted)] border border-white/10 hover:bg-white/[0.05]",
                  )}
                >
                  {ms / 1000}s
                </button>
              );
            })}
          </div>
        </div>
      ) : null}
    </>
  );
}
