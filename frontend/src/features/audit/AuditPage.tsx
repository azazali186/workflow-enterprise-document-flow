import { ScrollText, Search } from 'lucide-react';
import { PageHeader } from '@/components/ui/Card';
import { Select } from '@/components/ui/Field';
import { Badge, statusTone } from '@/components/ui/Badge';
import { DataTable, PaginationBar, type Column } from '@/components/ui/DataTable';
import { usePaginatedQuery } from '@/hooks/usePaginatedQuery';
import { auditService } from '@/services/audit.service';
import { formatDateTime, humanize } from '@/lib/format';
import type { AuditLog } from '@/types/entities';

export function AuditPage() {
  const table = usePaginatedQuery<AuditLog>(['audit-logs'], (req, signal) => auditService.list(req, signal), {
    defaultSortBy: 'created_at',
  });

  const columns: Column<AuditLog>[] = [
    {
      key: 'action',
      header: 'Action',
      cell: (l) => <Badge tone={statusTone(l.action)}>{l.action}</Badge>,
    },
    {
      key: 'entity',
      header: 'Entity',
      cell: (l) => (
        <div>
          <p className="font-medium text-ink-900">{humanize(l.entity)}</p>
          {l.entity_id && <p className="text-xs tabular text-ink-400">#{l.entity_id.slice(0, 8)}</p>}
        </div>
      ),
    },
    {
      key: 'actor',
      header: 'Actor',
      cell: (l) => (
        <div>
          <p className="text-[13px] font-medium text-ink-800">{l.actor_email || 'System'}</p>
          {l.ip_address && <p className="text-xs tabular text-ink-400">{l.ip_address}</p>}
        </div>
      ),
    },
    {
      key: 'when',
      header: 'When',
      sortKey: 'created_at',
      cell: (l) => <span className="text-[13px] tabular text-ink-500">{formatDateTime(l.created_at)}</span>,
    },
  ];

  return (
    <div className="space-y-5 animate-fade-up">
      <PageHeader
        title="Audit logs"
        description="Every important action, recorded with its actor."
        actions={
          <Badge tone="neutral">
            <ScrollText className="mr-1 size-3.5" />
            {table.total.toLocaleString()} events
          </Badge>
        }
      />

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1 sm:max-w-sm">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-300" aria-hidden />
          <input
            value={table.search}
            onChange={(e) => table.setSearch(e.target.value)}
            placeholder="Search by actor or entity…"
            className="h-9.5 w-full rounded-lg border border-ink-200 bg-white pl-9 pr-3 text-sm placeholder:text-ink-300 transition-all focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-500/30"
          />
        </div>
        <div className="w-full sm:w-44">
          <Select
            aria-label="Filter by action"
            value={String(table.filters.action ?? '')}
            onChange={(e) => table.setFilter('action', e.target.value)}
          >
            <option value="">All actions</option>
            <option value="create">create</option>
            <option value="update">update</option>
            <option value="delete">delete</option>
            <option value="login">login</option>
            <option value="logout">logout</option>
          </Select>
        </div>
      </div>

      <DataTable<AuditLog>
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
        emptyTitle="No audit events"
        emptyDescription="Actions will appear here as they happen."
        summary={table.summary}
        footer={<PaginationBar meta={table.meta} onNext={table.next} onPrev={table.prev} canPrev={table.canPrev} />}
      />
    </div>
  );
}
