import { useCallback, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useCursorPagination } from './useCursorPagination';
import { useDebounce } from './useDebounce';
import type { ListRequest, PageResult } from '@/types/api';

interface Options {
  limit?: number;
  defaultSortBy?: string;
  defaultSortDir?: 'asc' | 'desc';
  defaultFilters?: Record<string, unknown>;
  enabled?: boolean;
}

/**
 * Standard list-endpoint hook: debounced search, sort, filters, and cursor
 * pagination wired to React Query. Returns everything a page needs.
 */
export function usePaginatedQuery<T>(
  queryKey: readonly unknown[],
  fetcher: (req: ListRequest, signal?: AbortSignal) => Promise<PageResult<T>>,
  options: Options = {},
) {
  const limit = options.limit ?? 20;
  const { cursor, next, prev, reset, canPrev, applyResult, summary } = useCursorPagination(limit);

  const [search, setSearch] = useState('');
  const debouncedSearch = useDebounce(search);
  const [sortBy, setSortBy] = useState(options.defaultSortBy ?? 'created_at');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>(options.defaultSortDir ?? 'desc');
  const [filters, setFilters] = useState<Record<string, unknown>>(options.defaultFilters ?? {});

  const query = useQuery({
    queryKey: [...queryKey, { cursor, limit, sortBy, sortDir, search: debouncedSearch, filters }],
    queryFn: ({ signal }) =>
      fetcher(
        {
          cursor: cursor || undefined,
          limit,
          sort_by: sortBy,
          sort_dir: sortDir,
          search: debouncedSearch || undefined,
          filters,
        },
        signal,
      ),
    enabled: options.enabled !== false,
    placeholderData: (prev) => prev,
  });

  const meta = query.data?.pagination;

  // Record total from the server for the summary window math.
  useMemo(() => {
    if (query.data) {
      applyResult({ pagination: query.data.pagination });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query.data]);

  const toggleSort = useCallback(
    (key: string) => {
      setSortBy((prevBy) => {
        setSortDir((prevDir) => (prevBy !== key ? 'asc' : prevDir === 'asc' ? 'desc' : 'asc'));
        return key;
      });
      reset();
    },
    [reset],
  );

  const setFilter = useCallback(
    (key: string, value: unknown) => {
      setFilters((prev) => {
        const next = { ...prev };
        if (value === '' || value === null || value === undefined) delete next[key];
        else next[key] = value;
        return next;
      });
      reset();
    },
    [reset],
  );

  const changeSearch = useCallback(
    (value: string) => {
      setSearch(value);
      reset();
    },
    [reset],
  );

  const goNext = useCallback(() => next(query.data?.pagination.next_cursor), [next, query.data]);
  const goPrev = useCallback(() => prev(), [prev]);

  return {
    rows: query.data?.items,
    meta,
    summary,
    total: query.data?.pagination.total_count ?? 0,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
    search,
    setSearch: changeSearch,
    sortBy,
    sortDir,
    toggleSort,
    filters,
    setFilter,
    next: goNext,
    prev: goPrev,
    reset,
    canPrev,
    canNext: query.data?.pagination.has_more ?? false,
  };
}
