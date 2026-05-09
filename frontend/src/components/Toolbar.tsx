import { ArrowLeft, ArrowRight, Plus, Search } from "lucide-react";
import { useProjects } from "../store";
import { cn } from "../lib/cn";

interface ToolbarProps {
  title?: string;
  onOpenPalette: () => void;
  onAdd: () => void;
}

export function Toolbar({ title, onOpenPalette, onAdd }: ToolbarProps) {
  const { canBack, canForward, back, forward } = useProjects();

  return (
    <div className="relative flex items-center h-full px-3 gap-2">
      {/* Left: nav + title */}
      <div className="no-drag flex items-center gap-2">
        <Pill>
          <PillBtn disabled={!canBack} onClick={back}>
            <ArrowLeft className="size-4" />
          </PillBtn>
          <PillBtn disabled={!canForward} onClick={forward}>
            <ArrowRight className="size-4" />
          </PillBtn>
        </Pill>

        <span className="text-sm font-semibold text-[var(--orbit-text)] select-none truncate max-w-44">
          {title}
        </span>
      </div>

      {/* Center: search */}
      <div className="absolute left-1/2 -translate-x-1/2 no-drag w-96 max-w-[50%]">
        <SearchTrigger onClick={onOpenPalette} />
      </div>

      {/* Right: add */}
      <div className="no-drag ml-auto">
        <Pill>
          <PillBtn onClick={onAdd}>
            <Plus className="size-4" />
          </PillBtn>
        </Pill>
      </div>
    </div>
  );
}

// ─── primitives ──────────────────────────────────────────────────────────────

function Pill({ children }: { children: React.ReactNode }) {
  return (
    <div className="inline-flex items-center rounded-full bg-white/9 border border-white/11 overflow-hidden shadow-[0_4px_14px_rgba(0,0,0,0.35)]">
      {children}
    </div>
  );
}

function PillBtn({
  children,
  onClick,
  disabled,
}: {
  children: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "no-drag size-10 flex items-center justify-center transition-colors outline-none",
        "text-white/55",
        !disabled && "hover:bg-white/10 hover:text-white/85",
        disabled && "opacity-25 cursor-default",
      )}
    >
      {children}
    </button>
  );
}

// ─── search trigger ───────────────────────────────────────────────────────────

function SearchTrigger({ onClick }: { onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="group w-full flex items-center h-10 rounded-full bg-white/6 border border-white/10 hover:bg-white/9 hover:border-white/14 shadow-[0_4px_14px_rgba(0,0,0,0.3)] transition-colors text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <Search className="ml-3.5 size-4 text-[var(--orbit-muted)] shrink-0" />
      <span className="flex-1 px-2 text-sm text-[var(--orbit-muted)]">
        Search...
      </span>
      <kbd className="mr-3 inline-flex items-center gap-0.5 text-xs text-white/70 font-semibold tracking-wide">
        <span className="text-sm leading-none">⌘</span>
        <span className="leading-none">K</span>
      </kbd>
    </button>
  );
}
