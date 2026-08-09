import { Link } from 'react-router-dom';
import { Archive } from 'lucide-react';
import { cn } from '@/lib/cn';

/** The DocuFlow brand mark: archive glyph on the indigo signature square. */
export function LogoMark({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        'flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary-600 text-white shadow-sm',
        className,
      )}
      aria-hidden
    >
      <Archive className="size-4.5" />
    </span>
  );
}

interface LogoProps {
  /** Color of the wordmark text. */
  tone?: 'dark' | 'light';
  className?: string;
}

export function Logo({ tone = 'dark', className }: LogoProps) {
  return (
    <Link
      to="/"
      aria-label="DocuFlow home"
      className={cn('flex items-center gap-2.5', className)}
    >
      <LogoMark />
      <span
        className={cn(
          'font-display text-lg font-semibold tracking-tight',
          tone === 'dark' ? 'text-ink-950' : 'text-white',
        )}
      >
        DocuFlow
      </span>
    </Link>
  );
}
