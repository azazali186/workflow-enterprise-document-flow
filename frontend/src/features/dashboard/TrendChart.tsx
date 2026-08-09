import { useMemo } from 'react';

interface TrendChartProps {
  data: Record<string, number>;
  height?: number;
}

/** Minimal dependency-free bar chart for daily trends. */
export function TrendChart({ data, height = 180 }: TrendChartProps) {
  const entries = useMemo(
    () =>
      Object.entries(data)
        .sort(([a], [b]) => a.localeCompare(b))
        .slice(-14),
    [data],
  );

  if (entries.length === 0) {
    return (
      <div className="flex h-40 items-center justify-center text-sm text-ink-400">
        No activity in this window yet.
      </div>
    );
  }

  const max = Math.max(1, ...entries.map(([, v]) => v));
  const barW = 100 / entries.length;

  return (
    <div className="w-full">
      <div className="flex items-end gap-1.5" style={{ height }}>
        {entries.map(([day, value]) => {
          const h = Math.round((value / max) * (height - 28)) + 4;
          const label = new Date(`${day}T00:00:00`).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
          return (
            <div key={day} className="group flex h-full flex-1 flex-col items-center justify-end gap-1.5">
              <span className="text-[10px] font-medium tabular text-ink-500 opacity-0 transition-opacity group-hover:opacity-100">
                {value}
              </span>
              <div
                className="w-full max-w-8 rounded-t-md bg-primary-600/80 transition-all duration-200 group-hover:bg-primary-600"
                style={{ height: `${h}px` }}
                title={`${label}: ${value}`}
              />
              <span className="truncate text-[10px] tabular text-ink-400" style={{ maxWidth: `${barW}%` }}>
                {label}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
