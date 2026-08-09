import { useEffect } from 'react';
import { CheckCircle2, Info, X, XCircle } from 'lucide-react';
import { createPortal } from 'react-dom';
import { useAppDispatch, useAppSelector } from '@/store';
import { dismissToast, type Toast as ToastItem } from '@/store/uiSlice';
import { cn } from '@/lib/cn';

const kindStyles: Record<ToastItem['kind'], string> = {
  success: 'border-success-500/25 text-success-600',
  error: 'border-danger-500/25 text-danger-600',
  info: 'border-primary-500/25 text-primary-600',
};

const kindIcons = {
  success: CheckCircle2,
  error: XCircle,
  info: Info,
};

export function Toaster() {
  const toasts = useAppSelector((s) => s.ui.toasts);
  const dispatch = useAppDispatch();

  return createPortal(
    <div className="pointer-events-none fixed inset-x-0 top-4 z-[60] flex flex-col items-center gap-2 px-4 sm:items-end sm:pr-6">
      {toasts.map((t) => (
        <ToastCard key={t.id} toast={t} onDismiss={() => dispatch(dismissToast(t.id))} />
      ))}
    </div>,
    document.body,
  );
}

function ToastCard({ toast, onDismiss }: { toast: ToastItem; onDismiss: () => void }) {
  const Icon = kindIcons[toast.kind];

  useEffect(() => {
    const t = setTimeout(onDismiss, 5000);
    return () => clearTimeout(t);
  }, [onDismiss]);

  return (
    <div
      role="status"
      className={cn(
        'pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-xl border bg-white p-3.5 shadow-pop animate-fade-up',
        kindStyles[toast.kind],
      )}
    >
      <Icon className="mt-0.5 size-4.5 shrink-0" aria-hidden />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-ink-900">{toast.title}</p>
        {toast.message && <p className="mt-0.5 text-[13px] leading-snug text-ink-500">{toast.message}</p>}
      </div>
      <button
        onClick={onDismiss}
        aria-label="Dismiss notification"
        className="rounded-md p-1 text-ink-300 transition-colors hover:bg-ink-100 hover:text-ink-600 cursor-pointer"
      >
        <X className="size-3.5" />
      </button>
    </div>
  );
}
