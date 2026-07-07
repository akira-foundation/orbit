import { useEffect, useState } from "react";
import { Check, ChevronDown, Layers, Play, Plus, Square, Trash2 } from "lucide-react";
import { useProjects } from "../store";
import { cn } from "../lib/cn";

export function SidebarGroups() {
  const {
    groups,
    projects,
    loadGroups,
    createGroup,
    deleteGroup,
    startGroup,
    stopGroup,
    addToGroup,
    removeFromGroup,
  } = useProjects();
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);

  useEffect(() => {
    loadGroups();
  }, [loadGroups]);

  const submit = async () => {
    const trimmed = name.trim();
    if (trimmed) await createGroup(trimmed);
    setName("");
    setCreating(false);
  };

  if (groups.length === 0 && !creating) {
    return (
      <div className="mt-5">
        <SectionHeader onAdd={() => setCreating(true)} />
      </div>
    );
  }

  return (
    <div className="mt-5">
      <SectionHeader onAdd={() => setCreating(true)} />
      <ul className="space-y-1">
        {creating && (
          <li className="px-3 py-1">
            <input
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") submit();
                if (e.key === "Escape") {
                  setName("");
                  setCreating(false);
                }
              }}
              onBlur={submit}
              placeholder="Group name"
              className="w-full h-7 px-2 rounded-md bg-white/5 border border-white/10 text-[12px] outline-none focus:border-[var(--orbit-accent-2)]/50 placeholder:text-[var(--orbit-subtle)]"
            />
          </li>
        )}
        {groups.map((g) => {
          const isOpen = expanded === g.id;
          return (
            <li key={g.id}>
              <div className="group/g h-[34px] px-3 rounded-md flex items-center gap-2 text-[13px] text-[var(--orbit-text)]/90 hover:bg-white/[0.06] transition">
                <button
                  onClick={() => setExpanded(isOpen ? null : g.id)}
                  className="flex items-center gap-2 flex-1 min-w-0 text-left"
                >
                  <ChevronDown
                    className={cn(
                      "h-3.5 w-3.5 shrink-0 text-[var(--orbit-muted)] transition-transform",
                      !isOpen && "-rotate-90",
                    )}
                  />
                  <Layers className="h-[15px] w-[15px] shrink-0 text-[var(--orbit-muted)]" />
                  <span className="truncate">{g.name}</span>
                  <span className="text-[10px] text-[var(--orbit-subtle)]">
                    {g.projectIds.length}
                  </span>
                </button>
                <button
                  onClick={() => startGroup(g.id)}
                  title="Start all"
                  className="opacity-0 group-hover/g:opacity-100 text-emerald-300 hover:bg-white/10 rounded p-0.5 transition"
                >
                  <Play className="h-3.5 w-3.5" />
                </button>
                <button
                  onClick={() => stopGroup(g.id)}
                  title="Stop all"
                  className="opacity-0 group-hover/g:opacity-100 text-rose-300 hover:bg-white/10 rounded p-0.5 transition"
                >
                  <Square className="h-3.5 w-3.5" />
                </button>
                <button
                  onClick={() => deleteGroup(g.id)}
                  title="Delete group"
                  className="opacity-0 group-hover/g:opacity-100 text-[var(--orbit-muted)] hover:text-rose-300 hover:bg-white/10 rounded p-0.5 transition"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
              {isOpen && (
                <ul className="ml-6 mt-0.5 space-y-0.5">
                  {projects.map((p) => {
                    const member = g.projectIds.includes(p.id);
                    return (
                      <li key={p.id}>
                        <button
                          onClick={() =>
                            member
                              ? removeFromGroup(g.id, p.id)
                              : addToGroup(g.id, p.id)
                          }
                          className="w-full h-7 px-2 rounded-md flex items-center gap-2 text-[12px] hover:bg-white/[0.05] transition"
                        >
                          <span
                            className={cn(
                              "size-3.5 shrink-0 rounded border flex items-center justify-center",
                              member
                                ? "bg-[var(--orbit-accent-2)]/20 border-[var(--orbit-accent-2)]/50"
                                : "border-white/15",
                            )}
                          >
                            {member && (
                              <Check className="size-2.5 text-[var(--orbit-accent-2)]" />
                            )}
                          </span>
                          <span className="truncate text-[var(--orbit-text)]/80">
                            {p.name}
                          </span>
                        </button>
                      </li>
                    );
                  })}
                  {projects.length === 0 && (
                    <li className="px-2 text-[11px] text-[var(--orbit-subtle)] italic">
                      No projects to add.
                    </li>
                  )}
                </ul>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}

function SectionHeader({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="flex items-center justify-between px-3 mb-1.5">
      <h3 className="text-[11px] font-semibold text-[var(--orbit-subtle)]">Groups</h3>
      <button
        onClick={onAdd}
        title="New group"
        className="text-[var(--orbit-subtle)] hover:text-[var(--orbit-text)] transition"
      >
        <Plus className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
