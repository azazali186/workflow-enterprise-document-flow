import { cn } from '@/lib/cn';

const tones = [
  'bg-primary-100 text-primary-700',
  'bg-success-50 text-success-600',
  'bg-warning-50 text-warning-600',
  'bg-sky-50 text-sky-700',
  'bg-ink-100 text-ink-700',
];

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase() ?? '')
    .join('') || '?';
}

export function Avatar({ name, size = 'md', className }: { name: string; size?: 'sm' | 'md'; className?: string }) {
  const idx = (name.charCodeAt(0) || 0) % tones.length;
  return (
    <div
      className={cn(
        'flex shrink-0 items-center justify-center rounded-full font-semibold',
        size === 'md' ? 'size-9 text-[13px]' : 'size-7 text-xs',
        tones[idx],
        className,
      )}
      aria-hidden
    >
      {initials(name)}
    </div>
  );
}
