import { SettingsBody } from "../components/SettingsDialog";

export function SettingsPage({
  autoAction,
  onAutoActionConsumed,
}: {
  autoAction?: "setup" | "reset" | null;
  onAutoActionConsumed?: () => void;
}) {
  return (
    <div className="h-full overflow-hidden p-6">
      <div className="mx-auto h-full w-full max-w-4xl rounded-2xl border border-[var(--orbit-border)] bg-white/[0.02] overflow-hidden">
        <SettingsBody
          autoAction={autoAction}
          onAutoActionConsumed={onAutoActionConsumed}
          visible
          className="h-full"
        />
      </div>
    </div>
  );
}
