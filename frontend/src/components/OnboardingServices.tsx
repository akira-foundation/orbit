import { useEffect, useState } from "react";
import { api } from "../api";
import type { ServiceInfo } from "../types";
import { Switch } from "./ui/switch";

const DONE_KEY = "orbit.onboarding.services";

export function OnboardingServices({ onDone }: { onDone: () => void }) {
  const [items, setItems] = useState<ServiceInfo[]>([]);
  const [picked, setPicked] = useState<Record<string, boolean>>({});
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api.listServices().then(setItems);
  }, []);

  function finish() {
    localStorage.setItem(DONE_KEY, "done");
    onDone();
  }

  async function confirm() {
    setBusy(true);
    try {
      for (const svc of items) {
        if (picked[svc.engine]) {
          await api.startService(svc.engine);
        }
      }
      finish();
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="w-[420px] rounded-2xl border border-white/10 bg-zinc-950 p-6">
        <h1 className="text-base font-semibold mb-1">Set up local services</h1>
        <p className="text-xs text-zinc-400 mb-4">
          Pick the services to download now. You can add more later from the
          Services page. Nothing is downloaded unless you choose it.
        </p>
        <div className="flex flex-col gap-2 mb-5">
          {items.map((svc) => (
            <div
              key={svc.engine}
              className="flex items-center justify-between rounded-lg border border-white/10 px-3 py-2"
            >
              <span className="text-sm">{svc.displayName}</span>
              <Switch
                checked={!!picked[svc.engine]}
                onCheckedChange={(checked) =>
                  setPicked((p) => ({ ...p, [svc.engine]: checked }))
                }
              />
            </div>
          ))}
        </div>
        <div className="flex justify-end gap-2">
          <button
            className="rounded-lg px-3 py-1.5 text-sm text-zinc-400 hover:text-zinc-200"
            onClick={finish}
          >
            Skip
          </button>
          <button
            className="rounded-lg bg-cyan-500/90 px-4 py-1.5 text-sm font-medium text-black hover:bg-cyan-400 disabled:opacity-50"
            disabled={busy}
            onClick={confirm}
          >
            {busy ? "Downloading…" : "Download selected"}
          </button>
        </div>
      </div>
    </div>
  );
}

export function onboardingPending(): boolean {
  return localStorage.getItem(DONE_KEY) !== "done";
}

export function resetOnboarding() {
  localStorage.removeItem(DONE_KEY);
}
