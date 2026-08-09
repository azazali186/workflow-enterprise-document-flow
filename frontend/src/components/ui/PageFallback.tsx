export function PageFallback() {
  return (
    <div className="flex min-h-dvh items-center justify-center bg-paper-50">
      <div className="flex flex-col items-center gap-3">
        <div className="size-8 animate-spin rounded-full border-2 border-ink-200 border-t-primary-600" aria-label="Loading" />
        <p className="text-xs text-ink-400">Loading…</p>
      </div>
    </div>
  );
}
