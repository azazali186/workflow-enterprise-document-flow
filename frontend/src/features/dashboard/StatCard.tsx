import type { LucideIcon } from 'lucide-react';
import { Card } from '@/components/ui/Card';
import { cn } from '@/lib/cn';

interface StatCardProps {
  label: string;
  value: string;
  sub?: string;
  icon: LucideIcon;
  tone?: 'primary' | 'success' | 'warning' | 'danger' | 'neutral';
}

const toneIcon = {
  primary: 'bg-primary-50 text-primary-600',
  success: 'bg-success-50 text-success-600',
  warning: 'bg-warning-50 text-warning-600',
  danger: 'bg-danger-50 text-danger-600',
  neutral: 'bg-ink-100 text-ink-600',
};

export function StatCard({ label, value, sub, icon: Icon, tone = 'neutral' }: StatCardProps) {
  return (
    <Card className="p-5 transition-shadow duration-200 hover:shadow-pop">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-[13px] font-medium text-ink-500">{label}</p>
          <p className="mt-2 font-display text-[28px] font-semibold leading-none tracking-tight text-ink-950 tabular">
            {value}
          </p>
          {sub && <p className="mt-2 text-xs text-ink-400">{sub}</p>}
        </div>
        <div className={cn('flex size-10 shrink-0 items-center justify-center rounded-xl', toneIcon[tone])}>
          <Icon className="size-5" aria-hidden />
        </div>
      </div>
    </Card>
  );
}
