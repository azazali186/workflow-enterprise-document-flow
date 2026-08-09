import { useCallback, useMemo, useState } from 'react';
import type { PaginationMeta } from '@/types/api';

interface PaginationState {
  /** Cursor chain; index points at the current page's cursor. */
  history: string[];
  index: number;
}

/**
 * Server-side cursor (keyset) pagination with a client history stack so users
 * can page forward and back. Each page is `limit` items (except the last), so
 * the visible window for page N is N*limit+1 .. (N+1)*limit, capped by the
 * server-reported total. The summary line ("Showing 11–20 of 245 users")
 * is derived from that window.
 */
export function useCursorPagination(limit: number) {
  const [state, setState] = useState<PaginationState>({ history: [''], index: 0 });
  const [total, setTotal] = useState(0);

  const cursor = state.history[state.index] ?? '';
  const canPrev = state.index > 0;

  const applyResult = useCallback((result: { pagination: PaginationMeta }) => {
    setTotal(result.pagination.total_count ?? 0);
  }, []);

  const next = useCallback(
    (nextCursor: string | undefined) => {
      if (!nextCursor) return;
      setState((s) => {
        const history = s.history.slice(0, s.index + 1);
        history.push(nextCursor);
        return { history, index: history.length - 1 };
      });
    },
    [],
  );

  const prev = useCallback(() => {
    setState((s) => ({ ...s, index: Math.max(0, s.index - 1) }));
  }, []);

  const reset = useCallback(() => {
    setState({ history: [''], index: 0 });
    setTotal(0);
  }, []);

  const page = state.index;
  const start = page * limit + 1;
  const end = Math.min((page + 1) * limit, total);

  const summary = useMemo(() => {
    if (total === 0) return '';
    return `Showing ${start.toLocaleString()}–${end.toLocaleString()} of ${total.toLocaleString()}`;
  }, [start, end, total]);

  return {
    cursor,
    page,
    next,
    prev,
    reset,
    canPrev,
    applyResult,
    summary,
    total,
    limit,
  };
}
