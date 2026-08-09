import type { ReactNode } from 'react';
import { cn } from '@/lib/cn';

type Tone = 'neutral' | 'primary' | 'success' | 'warning' | 'danger' | 'info';

const toneClasses: Record<Tone, string> = {
  neutral: 'bg-ink-100 text-ink-600',
  primary: 'bg-primary-50 text-primary-700',
  success: 'bg-success-50 text-success-600',
  warning: 'bg-warning-50 text-warning-600',
  danger: 'bg-danger-50 text-danger-600',
  info: 'bg-sky-50 text-sky-700',
};

interface BadgeProps {
  tone?: Tone;
  children: ReactNode;
  dot?: boolean;
  className?: string;
}

export function Badge({ tone = 'neutral', children, dot, className }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium',
        toneClasses[tone],
        className,
      )}
    >
      {dot && <span className="size-1.5 rounded-full bg-current" aria-hidden />}
      {children}
    </span>
  );
}

/** Maps a document/user status string to a Badge tone. */
export function statusTone(status?: string | null): Tone {
  switch (status) {
    case 'active':
    case 'approved':
    case 'verified':
    case 'success':
    case 'stored':
      return 'success';
    case 'pending':
    case 'pending_verification':
    case 'in_progress':
    case 'locked':
      return 'warning';
    case 'rejected':
    case 'failure':
    case 'failed':
      return 'danger';
    case 'draft':
    case 'archived':
      return 'neutral';
    default:
      return 'info';
  }
}
