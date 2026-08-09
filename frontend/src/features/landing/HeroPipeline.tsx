import { BadgeCheck, FileText, History, ShieldCheck } from 'lucide-react';
import { cn } from '@/lib/cn';

const STAGES = [
  { label: 'Draft', icon: FileText, tone: 'text-ink-500' },
  { label: 'Verify', icon: BadgeCheck, tone: 'text-warning-600' },
  { label: 'Approve', icon: ShieldCheck, tone: 'text-primary-600' },
  { label: 'Archive', icon: History, tone: 'text-success-600' },
];

/**
 * The signature hero visual: a document card flowing through the four
 * lifecycle stages along an animated dashed track. Pure CSS/SVG — no image
 * assets, stays crisp on every screen.
 */
export function HeroPipeline() {
  return (
    <div className="relative mx-auto w-full max-w-3xl" aria-hidden>
      {/* Soft gradient halo behind the card */}
      <div className="absolute inset-x-8 -top-10 bottom-0 rounded-[3rem] bg-gradient-to-b from-primary-100/70 via-paper-50 to-transparent blur-2xl" />

      {/* Stage chips */}
      <div className="relative grid grid-cols-4 gap-2 sm:gap-4">
        {STAGES.map((stage, i) => (
          <div
            key={stage.label}
            className="flex flex-col items-center gap-2 text-center"
            style={{ transitionDelay: `${i * 90}ms` }}
          >
            <span
              className={cn(
                'flex size-9 items-center justify-center rounded-full border bg-white shadow-card sm:size-10',
                'border-ink-200/80',
              )}
            >
              <stage.icon className={cn('size-4 sm:size-4.5', stage.tone)} />
            </span>
            <span className="text-[10.5px] font-semibold uppercase tracking-[0.14em] text-ink-500 sm:text-[11px]">
              {stage.label}
            </span>
          </div>
        ))}
      </div>

      {/* Animated dashed connector */}
      <div className="absolute left-[12.5%] right-[12.5%] top-5 hidden sm:block" aria-hidden>
        <svg className="w-full" height="6" viewBox="0 0 600 6" preserveAspectRatio="none">
          <defs>
            <pattern id="pipeline-dash" width="18" height="6" patternUnits="userSpaceOnUse">
              <circle cx="3" cy="3" r="2.5" fill="var(--color-primary-300)" />
            </pattern>
          </defs>
          <line
            x1="0"
            y1="3"
            x2="600"
            y2="3"
            stroke="url(#pipeline-dash)"
            strokeWidth="4"
            strokeDasharray="0 12"
            className="dash-flow"
          />
        </svg>
      </div>

      {/* The document card */}
      <div className="relative z-10 mt-6 sm:mt-8">
        <div className="float-slow mx-auto w-full max-w-md rounded-2xl border border-ink-200/70 bg-white p-5 shadow-pop">
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <div className="flex size-10 items-center justify-center rounded-xl bg-paper-100 text-ink-500">
                <FileText className="size-5" aria-hidden />
              </div>
              <div>
                <p className="text-sm font-semibold text-ink-900">Q3 Financial Report</p>
                <p className="text-xs tabular text-ink-400">DOC-2026-0184 · v3</p>
              </div>
            </div>
            <span className="rounded-full bg-success-50 px-2.5 py-1 text-[11px] font-semibold text-success-600">
              Approved
            </span>
          </div>

          {/* Mini progress trail inside the card */}
          <div className="mt-4 flex items-center gap-1.5" aria-hidden>
            {[0, 1, 2, 3].map((i) => (
              <span key={i} className="h-1 flex-1 overflow-hidden rounded-full bg-ink-100">
                <span
                  className={cn(
                    'block h-full rounded-full',
                    i < 3 ? 'w-full bg-primary-500' : 'w-0',
                  )}
                />
              </span>
            ))}
          </div>
          <div className="mt-3 flex items-center justify-between text-[11px] text-ink-400">
            <span>Draft → Verified → Approved</span>
            <span className="tabular">24h 06m</span>
          </div>
        </div>
      </div>
    </div>
  );
}
