import { useMemo, type ReactNode } from 'react';
import { ArrowDown, ArrowUp, ChevronsUpDown } from 'lucide-react';
import { cn } from '@/lib/cn';
import type { PaginationMeta } from '@/types/api';
import { TableSkeleton } from './Skeleton';
import { EmptyState, ErrorState } from './States';

export interface Column<T> {
  key: string;
  header: string;
  /** Render cell content. */
  cell: (row: T) => ReactNode;
  /** Optional sortable column key sent to the API. */
  sortKey?: string;
  className?: string;
  headerClassName?: string;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  rows: T[] | undefined;
  meta?: PaginationMeta;
  loading?: boolean;
  error?: unknown;
  onRetry?: () => void;
  sortBy?: string;
  sortDir?: 'asc' | 'desc';
  onSort?: (key: string) => void;
  emptyTitle?: string;
  emptyDescription?: string;
  rowKey: (row: T) => string;
  onRowClick?: (row: T) => void;
  footer?: ReactNode;
  /** Full API range summary line, e.g. "Showing 11–20 of 245 users". */
  summary?: string;
}

export function DataTable<T>({
  columns,
  rows,
  meta,
  loading,
  error,
  onRetry,
  sortBy,
  sortDir,
  onSort,
  emptyTitle = 'Nothing here yet',
  emptyDescription,
  rowKey,
  onRowClick,
  footer,
  summary,
}: DataTableProps<T>) {
  const hasData = rows !== undefined && rows.length > 0;

  const summaryLine = useMemo(() => {
    if (summary) return summary;
    if (!meta || !rows || rows.length === 0) return null;
    return `Showing ${rows.length} of ${meta.total_count.toLocaleString()}`;
  }, [summary, meta, rows]);

  return (
    <div className="flex flex-col overflow-hidden rounded-xl border border-ink-200/70 bg-white shadow-card">
      {summaryLine && (
        <div className="flex items-center justify-between border-b border-ink-100 px-5 py-2.5">
          <p className="text-[13px] text-ink-500">{summaryLine}</p>
        </div>
      )}
      <div className="nice-scroll overflow-x-auto">
        <table className="w-full min-w-[640px] border-collapse text-left text-sm">
          <thead>
            <tr className="border-b border-ink-100">
              {columns.map((col) => (
                <th
                  key={col.key}
                  className={cn(
                    'whitespace-nowrap px-5 py-3 text-xs font-semibold uppercase tracking-wide text-ink-400',
                    col.headerClassName,
                  )}
                >
                  {col.sortKey && onSort ? (
                    <button
                      onClick={() => onSort(col.sortKey!)}
                      className={cn(
                        'group inline-flex items-center gap-1 uppercase tracking-wide transition-colors hover:text-ink-700 cursor-pointer',
                        sortBy === col.sortKey && 'text-primary-600',
                      )}
                    >
                      {col.header}
                      {sortBy === col.sortKey ? (
                        sortDir === 'asc' ? (
                          <ArrowUp className="size-3.5" />
                        ) : (
                          <ArrowDown className="size-3.5" />
                        )
                      ) : (
                        <ChevronsUpDown className="size-3.5 opacity-40 group-hover:opacity-70" />
                      )}
                    </button>
                  ) : (
                    col.header
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-ink-100/80">
            {loading && !hasData ? (
              <tr>
                <td colSpan={columns.length} className="p-0">
                  <TableSkeleton rows={6} columns={columns.length} />
                </td>
              </tr>
            ) : error && !hasData ? (
              <tr>
                <td colSpan={columns.length} className="p-0">
                  <ErrorState onRetry={onRetry} />
                </td>
              </tr>
            ) : !hasData ? (
              <tr>
                <td colSpan={columns.length} className="p-0">
                  <EmptyState title={emptyTitle} description={emptyDescription} />
                </td>
              </tr>
            ) : (
              rows!.map((row) => (
                <tr
                  key={rowKey(row)}
                  onClick={onRowClick ? () => onRowClick(row) : undefined}
                  className={cn(
                    'transition-colors duration-100',
                    onRowClick ? 'cursor-pointer hover:bg-paper-50' : 'hover:bg-paper-50/60',
                  )}
                >
                  {columns.map((col) => (
                    <td key={col.key} className={cn('px-5 py-3.5 align-middle text-ink-700', col.className)}>
                      {col.cell(row)}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
      {footer}
    </div>
  );
}

/** Pagination bar driven by the cursor meta returned by the API. */
export function PaginationBar({
  meta,
  onNext,
  onPrev,
  canPrev,
}: {
  meta?: PaginationMeta;
  onNext: () => void;
  onPrev: () => void;
  canPrev: boolean;
}) {
  if (!meta) return null;
  const hasMore = Boolean(meta.has_more);
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-t border-ink-100 bg-paper-50/50 px-5 py-3">
      <div />
      <div className="flex items-center gap-2">
        <button
          onClick={onPrev}
          disabled={!canPrev}
          className="rounded-lg border border-ink-200 bg-white px-3 py-1.5 text-[13px] font-medium text-ink-600 transition-colors hover:border-ink-300 hover:text-ink-900 disabled:cursor-not-allowed disabled:opacity-40"
        >
          ← Previous
        </button>
        <button
          onClick={onNext}
          disabled={!hasMore}
          className="rounded-lg border border-ink-200 bg-white px-3 py-1.5 text-[13px] font-medium text-ink-600 transition-colors hover:border-ink-300 hover:text-ink-900 disabled:cursor-not-allowed disabled:opacity-40"
        >
          Next →
        </button>
      </div>
    </div>
  );
}
