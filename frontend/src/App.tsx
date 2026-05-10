import { useCallback, useEffect, useMemo, useState } from "react";
import { Sidebar } from "./components/Sidebar";
import { Toolbar } from "./components/Toolbar";
import { StatusBar } from "./components/StatusBar";
import { AddProjectDialog } from "./components/AddProjectDialog";
import { CommandPalette } from "./components/CommandPalette";
import { Dashboard } from "./pages/Dashboard";
import { ProjectDetail } from "./pages/ProjectDetail";
import { useProjects } from "./store";
import { useWailsEvent } from "./hooks/useWailsEvent";
import { SetupBanner } from "./components/SetupBanner";
import type { RuntimeStatusEvent } from "./types";

const STATUS_EVENTS = [
  "runtime:starting",
  "runtime:running",
  "runtime:stopped",
  "runtime:error",
];

export function App() {
  const { selectedId, projects, filter, query, load, add, patchStatus } =
    useProjects();
  const [dialog, setDialog] = useState(false);
  const [palette, setPalette] = useState(false);

  useEffect(() => {
    load();
  }, [load]);

  const onRuntimeStatus = useCallback(
    (e: RuntimeStatusEvent) => patchStatus(e.projectId, e.snapshot.status),
    [patchStatus],
  );
  useWailsEvent<RuntimeStatusEvent>(STATUS_EVENTS, onRuntimeStatus);

  const selected = useMemo(
    () => projects.find((p) => p.id === selectedId) ?? null,
    [projects, selectedId],
  );

  const visibleCount = useMemo(() => {
    let xs = projects;
    if (filter !== "all") xs = xs.filter((p) => p.status === filter);
    if (query.trim()) {
      const q = query.toLowerCase();
      xs = xs.filter(
        (p) =>
          p.name.toLowerCase().includes(q) ||
          p.detectedFramework.toLowerCase().includes(q) ||
          p.localDomain.toLowerCase().includes(q),
      );
    }
    return xs.length;
  }, [projects, filter, query]);

  return (
    <div className="relative h-full">
      <div className="ambient" />

      <div className="drag absolute inset-0 z-0" />

      <div className="relative z-10 h-full flex gap-1.5 p-2">
        <aside className="island w-72 shrink-0 flex flex-col">
          <div className="drag h-12 shrink-0 pl-[68px]" />
          <Sidebar />
        </aside>

        <main className="flex-1 min-w-0 flex flex-col">
          <header className="drag h-12 shrink-0">
            <Toolbar
              onOpenPalette={() => setPalette(true)}
              onAdd={() => setDialog(true)}
            />
          </header>

          <SetupBanner />

          <div className="flex-1 min-h-0">
            {selected ? (
              <ProjectDetail id={selected.id} />
            ) : (
              <Dashboard onAdd={() => setDialog(true)} />
            )}
          </div>

          {!selected && <StatusBar count={visibleCount} />}
        </main>
      </div>

      <AddProjectDialog
        open={dialog}
        onClose={() => setDialog(false)}
        onCreated={(p) => add(p)}
      />

      <CommandPalette
        open={palette}
        onOpenChange={setPalette}
        onAdd={() => setDialog(true)}
      />
    </div>
  );
}
