import { Search } from 'lucide-react';
import { PageHeader } from '@/components/ui/Card';
import { Select } from '@/components/ui/Field';
import { Badge } from '@/components/ui/Badge';
import { DataTable, PaginationBar, type Column } from '@/components/ui/DataTable';
import { usePaginatedQuery } from '@/hooks/usePaginatedQuery';
import { loginLogService } from '@/services/audit.service';
import { formatDateTime } from '@/lib/format';
import type { LoginLog } from '@/types/entities';

export function LoginLogsPage() {
  const table = usePaginatedQuery<LoginLog>(['login-logs'], (req, signal) => loginLogService.list(req, signal), {
    defaultSortBy: 'created_at',
  });

  const columns: Column<LoginLog>[] = [
    {
      key: 'email',
      header: 'Email',
      sortKey: 'email',
      cell: (l) => <span className="font-medium text-ink-900">{l.email}</span>,
    },
    {
      key: 'status',
      header: 'Status',
      sortKey: 'status',
      cell: (l) => <Badge tone={l.status === 'success' ? 'success' : 'danger'} dot>{l.status}</Badge>,
    },
    {
      key: 'reason',
      header: 'Failure reason',
      cell: (l) => <span className="max-w-56 truncate text-[13px] text-ink-500">{l.failure_reason || '—'}</span>,
    },
    {
      key: 'ip',
      header: 'IP address',
      cell: (l) => <span className="text-[13px] tabular text-ink-500">{l.ip_address || '—'}</span>,
    },
    {
      key: 'at',
      header: 'When',
      sortKey: 'created_at',
      cell: (l) => <span className="text-[13px] tabular text-ink-500">{formatDateTime(l.created_at)}</span>,
    },
  ];

  return (
    <div className="space-y-5 animate-fade-up">
      <PageHeader
        title="Login logs"
        description="Every sign-in attempt, successful or not."
        actions={
          <div className="flex items-center gap-3">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-300" aria-hidden />
              <input
                value={table.search}
                onChange={(e) => table.setSearch(e.target.value)}
                placeholder="Search by email…"
                aria-label="Search login logs by email"
                className="h-9.5 w-full rounded-lg border border-ink-200 bg-white pl-9 pr-3 text-sm placeholder:text-ink-300 transition-all focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-500/30 sm:w-60"
              />
            </div>
            <div className="w-40">
              <Select
                aria-label="Filter by status"
                value={String(table.filters.status ?? '')}
                onChange={(e) => table.setFilter('status', e.target.value)}
              >
                <option value="">All</option>
                <option value="success">Success</option>
                <option value="failure">Failure</option>
              </Select>
            </div>
          </div>
        }
      />

      <DataTable<LoginLog>
        columns={columns}
        rows={table.rows}
        meta={table.meta}
        loading={table.isLoading}
        error={table.error}
        onRetry={() => void table.refetch()}
        sortBy={table.sortBy}
        sortDir={table.sortDir}
        onSort={table.toggleSort}
        rowKey={(l) => l.id}
        emptyTitle="No login attempts"
        emptyDescription="Sign-ins are recorded here as they happen."
        summary={table.summary}
        footer={<PaginationBar meta={table.meta} onNext={table.next} onPrev={table.prev} canPrev={table.canPrev} />}
      />
    </div>
  );
}
