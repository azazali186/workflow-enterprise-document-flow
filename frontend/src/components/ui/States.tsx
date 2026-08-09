import type { ReactNode } from 'react';
import { FileQuestion, Inbox, RefreshCw, TriangleAlert } from 'lucide-react';
import { Button } from './Button';

interface StateProps {
  title: string;
  description?: string;
  icon?: ReactNode;
  action?: ReactNode;
}

export function EmptyState({ title, description, icon, action }: StateProps) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 px-6 py-16 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-paper-100 text-ink-400">
        {icon ?? <Inbox className="size-6" aria-hidden />}
      </div>
      <h3 className="mt-2 text-sm font-semibold text-ink-800">{title}</h3>
      {description && <p className="max-w-sm text-sm text-ink-500">{description}</p>}
      {action && <div className="mt-3">{action}</div>}
    </div>
  );
}

export function ErrorState({
  title = 'Could not load this data',
  description,
  onRetry,
}: {
  title?: string;
  description?: string;
  onRetry?: () => void;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 px-6 py-16 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-danger-50 text-danger-500">
        <TriangleAlert className="size-6" aria-hidden />
      </div>
      <h3 className="mt-2 text-sm font-semibold text-ink-800">{title}</h3>
      {description && <p className="max-w-sm text-sm text-ink-500">{description}</p>}
      {onRetry && (
        <Button variant="outline" size="sm" className="mt-3" icon={<RefreshCw className="size-3.5" />} onClick={onRetry}>
          Try again
        </Button>
      )}
    </div>
  );
}

/** Fallback shown when a document/detail lookup fails or returns nothing. */
export function NotFoundState({ label = 'Item' }: { label?: string }) {
  return (
    <EmptyState
      icon={<FileQuestion className="size-6" />}
      title={`${label} not found`}
      description="It may have been deleted or you may not have access to it."
    />
  );
}
