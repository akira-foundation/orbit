import { useEffect, useState } from "react";
import {
  copilotApi,
  type CopilotConfigType,
  type CopilotProvider,
} from "../copilotApi";
import { cn } from "../lib/cn";
import { ToggleRow } from "./SettingsSections";

export function CopilotSettings() {
  const [cfg, setCfg] = useState<CopilotConfigType | null>(null);
  const [providers, setProviders] = useState<CopilotProvider[]>([]);

  useEffect(() => {
    copilotApi.config().then(setCfg);
    copilotApi.providers().then(setProviders);
  }, []);

  const save = async (next: CopilotConfigType) => {
    setCfg(next);
    await copilotApi.saveConfig(next);
  };

  return (
    <>
      <ToggleRow
        label="Orbit AI"
        description="Explain dev-server errors, diagnose slowness, and ask about a project from its Orbit AI tab. Runs your locally installed Claude Code or Codex CLI inside the project folder, so it can read project files and recent logs through your own subscription."
        checked={!!cfg && cfg.enabled}
        disabled={!cfg}
        onChange={(v) => cfg && save({ ...cfg, enabled: v })}
      />
      {cfg?.enabled ? (
        <div className="flex items-center justify-between gap-4 rounded-lg border border-white/10 bg-white/[0.03] p-4">
          <div className="min-w-0 space-y-1">
            <p className="text-[13px] font-medium">Provider</p>
            <p className="text-[11px] text-[var(--orbit-muted)] leading-relaxed">
              {providers.length === 0
                ? "No AI CLI detected. Install and sign in to the Claude Code CLI (claude) or the Codex CLI (codex)."
                : "Which installed CLI answers Orbit AI requests."}
            </p>
          </div>
          {providers.length > 0 ? (
            <div className="flex items-center gap-1 shrink-0">
              {[{ id: "auto", displayName: "Auto", version: "" }, ...providers].map(
                (p) => {
                  const active = (cfg.provider || "auto") === p.id;
                  return (
                    <button
                      key={p.id}
                      onClick={() => save({ ...cfg, provider: p.id })}
                      title={p.version ? `v${p.version}` : undefined}
                      className={cn(
                        "h-7 px-2.5 rounded-md text-[12px] transition",
                        active
                          ? "bg-[var(--orbit-accent-2)]/20 text-[var(--orbit-accent-2)] border border-[var(--orbit-accent-2)]/40"
                          : "text-[var(--orbit-muted)] border border-white/10 hover:bg-white/[0.05]",
                      )}
                    >
                      {p.displayName}
                    </button>
                  );
                },
              )}
            </div>
          ) : null}
        </div>
      ) : null}
    </>
  );
}
