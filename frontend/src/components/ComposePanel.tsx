import { useEffect, useState } from "react";
import { Boxes } from "lucide-react";
import { api } from "../api";
import type { ComposeInfo } from "../types";
import { cn } from "../lib/cn";

export function ComposePanel({ projectId }: { projectId: string }) {
  const [info, setInfo] = useState<ComposeInfo | null>(null);

  useEffect(() => {
    let alive = true;
    const load = () =>
      api
        .composeInfo(projectId)
        .then((i) => {
          if (alive) setInfo(i);
        })
        .catch(() => {});
    load();
    const t = setInterval(load, 3000);
    return () => {
      alive = false;
      clearInterval(t);
    };
  }, [projectId]);

  if (!info || !info.detected) return null;

  return (
    <div className="mt-4 rounded-lg border border-white/10 bg-white/[0.03]">
      <div className="px-4 pt-3 pb-2 border-b border-white/[0.06] flex items-center gap-2">
        <Boxes className="size-3.5 text-[var(--orbit-muted)]" />
        <p className="text-[13px] font-medium">Docker Compose</p>
        <code className="text-[11px] text-[var(--orbit-subtle)]">{info.file}</code>
      </div>
      {!info.available ? (
        <p className="px-4 py-3 text-[11px] text-amber-300/90 leading-relaxed">
          docker is not installed. Compose services are skipped when this
          project runs.
        </p>
      ) : info.services.length === 0 ? (
        <p className="px-4 py-3 text-[11px] text-[var(--orbit-subtle)]">
          No containers running. They start and stop with the project.
        </p>
      ) : (
        <ul className="divide-y divide-white/[0.04]">
          {info.services.map((s) => (
            <li
              key={s.name}
              className="px-4 py-2 flex items-center gap-2 text-[12px]"
            >
              <span
                className={cn(
                  "size-2 rounded-full shrink-0",
                  s.state === "running"
                    ? "bg-emerald-400 shadow-[0_0_8px_#34d399]"
                    : "bg-white/20",
                )}
              />
              <span className="font-medium truncate">{s.name}</span>
              <span className="text-[var(--orbit-subtle)] truncate">
                {s.status}
              </span>
              {s.ports ? (
                <span className="ml-auto font-mono text-[10.5px] text-[var(--orbit-muted)] truncate">
                  {s.ports}
                </span>
              ) : null}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
