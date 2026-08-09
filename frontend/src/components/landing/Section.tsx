import type { HTMLAttributes, ReactNode } from 'react';
import { cn } from '@/lib/cn';

/** Max-width wrapper shared by every landing section. */
export function Container({ className, ...rest }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('mx-auto w-full max-w-6xl px-5 sm:px-8', className)} {...rest} />;
}

/** Small uppercase kicker above section titles. */
export function Eyebrow({ children, tone = 'dark' }: { children: ReactNode; tone?: 'dark' | 'light' }) {
  return (
    <p
      className={cn(
        'text-[11px] font-semibold uppercase tracking-[0.18em]',
        tone === 'dark' ? 'text-primary-600' : 'text-primary-300',
      )}
    >
      {children}
    </p>
  );
}

interface SectionHeaderProps {
  eyebrow?: string;
  title: ReactNode;
  description?: ReactNode;
  tone?: 'dark' | 'light';
  center?: boolean;
  className?: string;
}

/** Consistent heading block used atop every landing section. */
export function SectionHeader({ eyebrow, title, description, tone = 'dark', center, className }: SectionHeaderProps) {
  return (
    <div
      className={cn(
        'max-w-2xl',
        center && 'mx-auto text-center',
        className,
      )}
      data-reveal
    >
      {eyebrow && <Eyebrow tone={tone}>{eyebrow}</Eyebrow>}
      <h2
        className={cn(
          'mt-3 font-display text-3xl font-semibold tracking-tight text-balance sm:text-4xl',
          tone === 'dark' ? 'text-ink-950' : 'text-white',
        )}
      >
        {title}
      </h2>
      {description && (
        <p
          className={cn(
            'mt-4 text-[15px] leading-relaxed sm:text-base',
            tone === 'dark' ? 'text-ink-500' : 'text-ink-300',
          )}
        >
          {description}
        </p>
      )}
    </div>
  );
}

/** Standard vertical rhythm for a landing section. */
export function Section({
  className,
  ...rest
}: HTMLAttributes<HTMLElement>) {
  return <section className={cn('py-16 sm:py-24', className)} {...rest} />;
}
