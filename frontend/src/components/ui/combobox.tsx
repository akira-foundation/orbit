import { useEffect, useRef, useState, type ReactNode } from "react";
import { ChevronsUpDown } from "lucide-react";
import { cn } from "@/lib/cn";
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
} from "./command";

export interface ComboboxOption {
  value: string;
  label: string;
  node?: ReactNode;
  keywords?: string;
  disabled?: boolean;
}

export function Combobox({
  options,
  value,
  onChange,
  placeholder = "Select…",
  searchPlaceholder = "Search…",
  emptyText = "No matches.",
  className,
}: {
  options: ComboboxOption[];
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  searchPlaceholder?: string;
  emptyText?: string;
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const wrap = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (wrap.current && !wrap.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, [open]);

  const selected = options.find((o) => o.value === value);

  return (
    <div ref={wrap} className={cn("relative", className)}>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="h-9 w-full px-3 inline-flex items-center justify-between gap-2 rounded-md bg-white/5 border border-white/10 text-[13px] outline-none hover:bg-white/[0.07] focus:border-[var(--orbit-accent-2)]/50 transition-colors"
      >
        <span className="min-w-0 flex items-center gap-2 truncate">
          {selected ? selected.node ?? selected.label : (
            <span className="text-[var(--orbit-muted)]">{placeholder}</span>
          )}
        </span>
        <ChevronsUpDown className="size-3.5 shrink-0 text-[var(--orbit-muted)]" />
      </button>

      {open && (
        <div className="absolute z-50 mt-1 w-full rounded-lg border border-white/10 bg-[rgba(28,28,34,0.98)] backdrop-blur-2xl shadow-xl overflow-hidden">
          <Command>
            <CommandInput placeholder={searchPlaceholder} className="text-[13px]" />
            <CommandList className="max-h-[240px]">
              <CommandEmpty>{emptyText}</CommandEmpty>
              {options.map((opt) => (
                <CommandItem
                  key={opt.value}
                  value={`${opt.label} ${opt.keywords ?? ""}`}
                  disabled={opt.disabled}
                  onSelect={() => {
                    onChange(opt.value);
                    setOpen(false);
                  }}
                >
                  {opt.node ?? opt.label}
                </CommandItem>
              ))}
            </CommandList>
          </Command>
        </div>
      )}
    </div>
  );
}
