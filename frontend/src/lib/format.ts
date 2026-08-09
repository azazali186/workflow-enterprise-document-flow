/** Formatting helpers — every function tolerates null/undefined/empty input. */

export function formatDate(value?: string | null, opts?: Intl.DateTimeFormatOptions): string {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleDateString('en-US', opts ?? { year: 'numeric', month: 'short', day: 'numeric' });
}

export function formatDateTime(value?: string | null): string {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

/** Compact relative time, e.g. "3h ago". */
export function timeAgo(value?: string | null): string {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return '—';
  const diff = Date.now() - d.getTime();
  const abs = Math.abs(diff);
  const units: [number, string][] = [
    [365 * 24 * 3600e3, 'y'],
    [30 * 24 * 3600e3, 'mo'],
    [24 * 3600e3, 'd'],
    [3600e3, 'h'],
    [60e3, 'm'],
  ];
  for (const [ms, label] of units) {
    if (abs >= ms) return `${Math.round(diff / ms)}${label} ago`;
  }
  return 'just now';
}

/** Human-readable byte sizes with safe fallback. */
export function formatBytes(value?: number | null): string {
  if (value === null || value === undefined || Number.isNaN(value) || value < 0) return '—';
  if (value === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  const n = value / 1024 ** i;
  return `${n >= 100 || i === 0 ? Math.round(n) : n.toFixed(1)} ${units[i]}`;
}

/** Locale number formatting with fallback. */
export function formatNumber(value?: number | null): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '—';
  return value.toLocaleString('en-US');
}

/** Humanize snake_case keys for labels, e.g. pending_verification → "Pending verification". */
export function humanize(key?: string | null): string {
  if (!key) return '—';
  return key
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase());
}
