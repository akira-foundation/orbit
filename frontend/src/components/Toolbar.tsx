import { Plus, Search } from "lucide-react";
import { cn } from "../lib/cn";

interface ToolbarProps {
  onOpenPalette: () => void;
  onAdd: () => void;
}

export function Toolbar({ onOpenPalette, onAdd }: ToolbarProps) {
  return (
    <div className="relative flex items-center justify-end h-full px-3 gap-2">
      <div className="no-drag flex items-center gap-2">
        <Pill>
          <PillBtn onClick={onOpenPalette} title="Search (⌘K)">
            <Search className="size-4" />
          </PillBtn>
        </Pill>
        <Pill>
          <PillBtn onClick={onAdd} title="Add project">
            <Plus className="size-4" />
          </PillBtn>
        </Pill>
      </div>
    </div>
  );
}

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
  title,
}: {
  children: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  title?: string;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      title={title}
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
