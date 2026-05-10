import { ToggleGroup, ToggleGroupItem } from "./ui/toggle-group";

export type MetricInterval = "1H" | "24H" | "7D" | "14D" | "30D";

export function getSinceTsForInterval(interval: MetricInterval): number {
  const now = Math.floor(Date.now() / 1000);
  const h = 3600;
  const d = 24 * h;
  switch (interval) {
    case "1H": return now - 1 * h;
    case "24H": return now - 24 * h;
    case "7D": return now - 7 * d;
    case "14D": return now - 14 * d;
    case "30D": return now - 30 * d;
    default: return now - 24 * h;
  }
}

interface MetricsFilterProps {
  value: MetricInterval;
  onChange: (v: MetricInterval) => void;
}

export function MetricsFilter({ value, onChange }: MetricsFilterProps) {
  return (
    <ToggleGroup
      type="single"
      value={value}
      onValueChange={(v) => {
        if (v) onChange(v as MetricInterval);
      }}
      className="bg-white/5 border border-white/10 rounded-lg p-1 h-8 shadow-sm overflow-hidden"
    >
      {(["1H", "24H", "7D", "14D", "30D"] as const).map((interval) => (
        <ToggleGroupItem
          key={interval}
          value={interval}
          className="h-6 px-3 text-[11px] font-medium rounded-md data-[state=on]:bg-white/10 data-[state=on]:text-white data-[state=off]:text-[var(--orbit-muted)] data-[state=off]:hover:text-white transition-colors border-0"
        >
          {interval}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  );
}
